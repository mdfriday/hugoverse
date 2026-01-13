package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Config Caddy 配置
type Config struct {
	AdminAPI       string `json:"admin_api"`       // Caddy Admin API 地址 (如 http://127.0.0.1:2019)
	ConfigPath     string `json:"config_path"`     // Caddy 配置文件路径
	BinaryPath     string `json:"binary_path"`     // Caddy 二进制文件路径
	DefaultBackend string `json:"default_backend"` // 默认后端服务地址 (如 127.0.0.1:1314)
	CouchDBBackend string `json:"couchdb_backend"` // CouchDB 服务地址 (如 127.0.0.1:5984)
	CoreDomain     string `json:"core_domain"`     // 核心域名 (如 mdfriday.site)
	PidFile        string `json:"pid_file"`        // PID 文件路径 (用于后台运行)
	LogFile        string `json:"log_file"`        // 日志文件路径
	DNSPodToken    string `json:"dnspod_token"`    // 腾讯云 DNS API Token (格式: SecretId,SecretKey，用于 DNS-01 challenge)
	ServerIP       string `json:"server_ip"`       // 服务器公网 IP (用于域名检查)
}

// Client Caddy HTTP 客户端
type Client struct {
	config     *Config
	httpClient *http.Client
	cmd        *exec.Cmd      // 用于管理 Caddy 进程
	checker    *DomainChecker // 域名检查器
}

// NewClient 创建 Caddy 客户端
func NewClient(config *Config) *Client {
	// 设置默认值
	if config.AdminAPI == "" {
		config.AdminAPI = "http://127.0.0.1:2019"
	}
	if config.BinaryPath == "" {
		config.BinaryPath = "caddy"
	}
	if config.DefaultBackend == "" {
		config.DefaultBackend = "127.0.0.1:1314"
	}
	if config.CouchDBBackend == "" {
		config.CouchDBBackend = "127.0.0.1:5984"
	}
	if config.CoreDomain == "" {
		config.CoreDomain = "mdfriday.com"
	}
	if config.PidFile == "" {
		config.PidFile = "/tmp/caddy.pid"
	}
	if config.LogFile == "" {
		config.LogFile = "/tmp/caddy.log"
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 如果配置了 ServerIP，初始化域名检查器
	if config.ServerIP != "" {
		client.checker = NewDomainChecker(config.ServerIP)
	}

	return client
}

// CaddyConfig Caddy 完整配置结构
type CaddyConfig struct {
	Admin *AdminConfig `json:"admin"`
	Apps  *AppsConfig  `json:"apps"`
}

// AdminConfig Caddy Admin 配置
type AdminConfig struct {
	Listen string `json:"listen"`
}

// AppsConfig Caddy Apps 配置
type AppsConfig struct {
	HTTP *HTTPConfig `json:"http"`
	TLS  *TLSConfig  `json:"tls,omitempty"`
}

// HTTPConfig HTTP App 配置
type HTTPConfig struct {
	Servers map[string]*ServerConfig `json:"servers"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Listen    []string         `json:"listen"`
	Routes    []Route          `json:"routes"`
	AutoHTTPS *AutoHTTPSConfig `json:"automatic_https,omitempty"` // 自动 HTTPS 配置
}

// AutoHTTPSConfig 自动 HTTPS 配置
type AutoHTTPSConfig struct {
	Disable bool `json:"disable,omitempty"` // 禁用自动 HTTPS
}

// Route 路由配置
type Route struct {
	ID     string         `json:"@id,omitempty"`
	Match  []MatchHost    `json:"match"`
	Handle []HandleConfig `json:"handle"`
}

// MatchHost 域名匹配
type MatchHost struct {
	Host []string `json:"host"`
}

// HandleConfig 处理器配置
type HandleConfig struct {
	Handler    string     `json:"handler"`
	Upstreams  []Upstream `json:"upstreams,omitempty"`   // for reverse_proxy
	Root       string     `json:"root,omitempty"`        // for file_server
	StatusCode int        `json:"status_code,omitempty"` // for static_response
}

// Upstream 上游服务器
type Upstream struct {
	Dial string `json:"dial"`
}

// CertificateInfo SSL 证书信息
type CertificateInfo struct {
	Domain    string    `json:"domain"`
	Status    string    `json:"status"` // "issued", "pending", "failed"
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	Issuer    string    `json:"issuer"`
	Error     string    `json:"error,omitempty"`
}

// StartServer 启动 Caddy 服务器
// 使用默认配置启动，包含核心域名和 CouchDB 的反向代理
func (c *Client) StartServer() error {
	// 判断是否为开发环境（localhost 或 127.0.0.1）
	isDev := c.config.CoreDomain == "localhost" || c.config.CoreDomain == "127.0.0.1"

	// 开发环境配置
	var serverConfig *ServerConfig
	if isDev {
		// 开发环境：只监听 HTTP 端口，禁用自动 HTTPS
		serverConfig = &ServerConfig{
			Listen: []string{":8080"}, // 使用非特权端口，不需要 sudo
			Routes: []Route{
				{
					ID: "core-localhost",
					Match: []MatchHost{
						{Host: []string{c.config.CoreDomain}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.DefaultBackend},
							},
						},
					},
				},
				{
					ID: "couchdb-localhost",
					Match: []MatchHost{
						{Host: []string{fmt.Sprintf("cdb.%s", c.config.CoreDomain)}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.CouchDBBackend},
							},
						},
					},
				},
			},
			AutoHTTPS: &AutoHTTPSConfig{
				Disable: true, // 开发环境禁用自动 HTTPS
			},
		}
	} else {
		// 生产环境：监听 80 和 443，启用自动 HTTPS
		serverConfig = &ServerConfig{
			Listen: []string{":80", ":443"},
			Routes: []Route{
				{
					ID: fmt.Sprintf("core-%s", c.config.CoreDomain),
					Match: []MatchHost{
						{Host: []string{c.config.CoreDomain}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.DefaultBackend},
							},
						},
					},
				},
				{
					ID: fmt.Sprintf("couchdb-%s", c.config.CoreDomain),
					Match: []MatchHost{
						{Host: []string{fmt.Sprintf("cdb.%s", c.config.CoreDomain)}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.CouchDBBackend},
							},
						},
					},
				},
			},
		}
	}

	// 生成配置
	config := &CaddyConfig{
		Admin: &AdminConfig{
			Listen: "127.0.0.1:2019",
		},
		Apps: &AppsConfig{
			HTTP: &HTTPConfig{
				Servers: map[string]*ServerConfig{
					"main": serverConfig,
				},
			},
		},
	}

	// 如果指定了配置文件路径，保存到文件
	if c.config.ConfigPath != "" {
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(c.config.ConfigPath, configJSON, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		// 使用配置文件启动
		c.cmd = exec.Command(c.config.BinaryPath, "run", "--config", c.config.ConfigPath)
	} else {
		// 通过 stdin 传递配置
		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		c.cmd = exec.Command(c.config.BinaryPath, "run", "--config", "-")
		c.cmd.Stdin = bytes.NewReader(configJSON)
	}

	// 设置输出
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr

	// 启动进程
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Caddy: %w", err)
	}

	// 等待服务器就绪（开发环境可能需要更长时间）
	waitTime := 3 * time.Second
	if isDev {
		waitTime = 5 * time.Second
	}
	time.Sleep(waitTime)

	// 验证服务器是否运行（增加重试次数）
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if err := c.Ping(); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Caddy started but Admin API not responding after %d retries", maxRetries)
}

// StartServerBackground 后台启动 Caddy 服务器
// 启动后立即返回，将 PID 写入 PidFile
func (c *Client) StartServerBackground() error {
	// 判断是否已经在运行
	if c.IsRunning() {
		return fmt.Errorf("Caddy is already running (PID file exists: %s)", c.config.PidFile)
	}

	// 配置文件路径
	configFile := c.config.ConfigPath
	if configFile == "" {
		configFile = "/tmp/caddy-config.json"
	}

	// 检查是否使用现有配置文件
	var useExistingConfig bool
	if c.config.ConfigPath != "" && c.config.ConfigPath != "/tmp/caddy-config.json" {
		// 用户指定了配置文件，检查是否存在
		if _, err := os.Stat(c.config.ConfigPath); err == nil {
			useExistingConfig = true
		}
	}

	// 如果不使用现有配置，生成新配置
	if !useExistingConfig {
		// 判断是否为开发环境
		isDev := c.config.CoreDomain == "localhost" || c.config.CoreDomain == "127.0.0.1"

		// 开发环境配置
		var serverConfig *ServerConfig
		if isDev {
			serverConfig = &ServerConfig{
				Listen: []string{":8080"},
				Routes: []Route{
					{
						ID: "core-localhost",
						Match: []MatchHost{
							{Host: []string{c.config.CoreDomain}},
						},
						Handle: []HandleConfig{
							{
								Handler: "reverse_proxy",
								Upstreams: []Upstream{
									{Dial: c.config.DefaultBackend},
								},
							},
						},
					},
					{
						ID: "couchdb-localhost",
						Match: []MatchHost{
							{Host: []string{fmt.Sprintf("cdb.%s", c.config.CoreDomain)}},
						},
						Handle: []HandleConfig{
							{
								Handler: "reverse_proxy",
								Upstreams: []Upstream{
									{Dial: c.config.CouchDBBackend},
								},
							},
						},
					},
				},
				AutoHTTPS: &AutoHTTPSConfig{
					Disable: true,
				},
			}
		} else {
			// 生产环境路由配置
			routes := []Route{
				{
					ID: fmt.Sprintf("core-app.%s", c.config.CoreDomain),
					Match: []MatchHost{
						{Host: []string{fmt.Sprintf("app.%s", c.config.CoreDomain)}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.DefaultBackend},
							},
						},
					},
				},
				{
					ID: fmt.Sprintf("core-cdb.%s", c.config.CoreDomain),
					Match: []MatchHost{
						{Host: []string{fmt.Sprintf("cdb.%s", c.config.CoreDomain)}},
					},
					Handle: []HandleConfig{
						{
							Handler: "reverse_proxy",
							Upstreams: []Upstream{
								{Dial: c.config.CouchDBBackend},
							},
						},
					},
				},
			}

			// 如果配置了 DNSPodToken，添加通配符占位路由（触发通配符证书申请）
			// 这个路由放在最后，作为 fallback，返回 404
			// 所有 *.mdfriday.com 的子域名会复用这个通配符证书
			if c.config.DNSPodToken != "" {
				routes = append(routes, Route{
					ID: fmt.Sprintf("wildcard-%s", c.config.CoreDomain),
					Match: []MatchHost{
						{Host: []string{fmt.Sprintf("*.%s", c.config.CoreDomain)}},
					},
					Handle: []HandleConfig{
						{
							Handler:    "static_response",
							StatusCode: 404,
						},
					},
				})
			}

			serverConfig = &ServerConfig{
				Listen: []string{":80", ":443"},
				Routes: routes,
			}
		}

		// 生成配置
		config := &CaddyConfig{
			Admin: &AdminConfig{
				Listen: "127.0.0.1:2019",
			},
			Apps: &AppsConfig{
				HTTP: &HTTPConfig{
					Servers: map[string]*ServerConfig{
						"main": serverConfig,
					},
				},
			},
		}

		// 生产环境：添加平台域名的 Wildcard TLS 配置
		// 这样所有 subdomain（如 user123.mdfriday.com）都会使用 Wildcard 证书
		// 凭据直接在配置中传递（环境变量方式在某些 Caddy 版本不工作）
		if !isDev && c.config.DNSPodToken != "" {
			secretID, secretKey := c.parseDNSPodToken()
			if secretID != "" && secretKey != "" {
				tlsConfig := GeneratePlatformTLSConfig(c.config.CoreDomain, secretID, secretKey)
				if tlsConfig != nil {
					config.Apps.TLS = tlsConfig
				}
			}
		}

		// 配置文件
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		// 保存配置到文件
		if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	// 打开日志文件
	logFile, err := os.OpenFile(c.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// 启动后台进程
	cmd := exec.Command(c.config.BinaryPath, "run", "--config", configFile)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// 设置环境变量（继承当前进程的环境变量）
	cmd.Env = os.Environ()

	// 如果配置了 DNSPodToken，通过环境变量传递给 Caddy（用于 DNS-01 challenge）
	if c.config.DNSPodToken != "" {
		secretID, secretKey := c.parseDNSPodToken()
		if secretID != "" && secretKey != "" {
			cmd.Env = append(cmd.Env, "TENCENTCLOUD_SECRET_ID="+secretID)
			cmd.Env = append(cmd.Env, "TENCENTCLOUD_SECRET_KEY="+secretKey)
		}
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Caddy: %w", err)
	}

	// 写入 PID 文件
	pid := cmd.Process.Pid
	if err := os.WriteFile(c.config.PidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// 等待服务器就绪
	time.Sleep(2 * time.Second)

	// 验证服务器是否运行
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		if err := c.Ping(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("Caddy started but Admin API not responding after %d retries", maxRetries)
}

// IsRunning 检查 Caddy 是否正在运行
func (c *Client) IsRunning() bool {
	// 检查 PID 文件是否存在
	pidBytes, err := os.ReadFile(c.config.PidFile)
	if err != nil {
		return false
	}

	// 解析 PID
	var pid int
	if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err != nil {
		return false
	}

	// 检查进程是否存在
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 在 Unix 系统上，发送信号 0 来检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPID 获取 Caddy 进程的 PID
func (c *Client) GetPID() (int, error) {
	pidBytes, err := os.ReadFile(c.config.PidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err != nil {
		return 0, fmt.Errorf("failed to parse PID: %w", err)
	}

	return pid, nil
}

// AddStaticSite 动态添加自定义域名的静态站点
// domain: 自定义域名 (如 example.com)
// sitePath: 静态站点文件路径 (如 /web/sites/example-com)
// A     mdfriday.com        →  <server-ip>
// A     *.mdfriday.com      →  <server-ip>
//
// 注意：为确保具体域名优先于通配符匹配，采用以下策略：
// 1. 删除通配符 route
// 2. 添加新 route
// 3. 重新添加通配符 route（保持在最后）
func (c *Client) AddStaticSite(domain, sitePath string) error {
	route := Route{
		ID: fmt.Sprintf("site-%s", sanitizeDomainForID(domain)),
		Match: []MatchHost{
			{Host: []string{domain}},
		},
		Handle: []HandleConfig{
			{
				Handler: "file_server",
				Root:    sitePath,
			},
		},
	}

	// 1. 获取并删除通配符 route（如果存在）
	wildcardRoute, err := c.getAndRemoveWildcardRoute()
	if err != nil {
		// 忽略错误，继续添加新 route
		wildcardRoute = nil
	}

	// 2. 添加新 route
	body, err := json.Marshal(route)
	if err != nil {
		// 如果失败，尝试恢复通配符 route
		if wildcardRoute != nil {
			if err := c.addRoute(*wildcardRoute); err != nil {
				// 通配符添加失败不影响主流程，只记录日志
				// 可以通过后续操作恢复
				fmt.Printf("failed to restore wildcard route: %v\n", err)
			}
		}
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	url := fmt.Sprintf("%s/config/apps/http/servers/main/routes", c.config.AdminAPI)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		if wildcardRoute != nil {
			if err := c.addRoute(*wildcardRoute); err != nil {
				// 通配符添加失败不影响主流程，只记录日志
				// 可以通过后续操作恢复
				fmt.Printf("failed to restore wildcard route: %v\n", err)
			}
		}
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if wildcardRoute != nil {
			if err := c.addRoute(*wildcardRoute); err != nil {
				// 通配符添加失败不影响主流程，只记录日志
				// 可以通过后续操作恢复
				fmt.Printf("failed to restore wildcard route: %v\n", err)
			}
		}
		return fmt.Errorf("failed to add route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		if wildcardRoute != nil {
			if err := c.addRoute(*wildcardRoute); err != nil {
				// 通配符添加失败不影响主流程，只记录日志
				// 可以通过后续操作恢复
				fmt.Printf("failed to restore wildcard route: %v\n", err)
			}
		}
		return fmt.Errorf("failed to add route (status %d): %s", resp.StatusCode, string(respBody))
	}

	// 3. 重新添加通配符 route（保持在最后）
	if wildcardRoute != nil {
		if err := c.addRoute(*wildcardRoute); err != nil {
			// 通配符添加失败不影响主流程，只记录日志
			// 可以通过后续操作恢复
			fmt.Printf("failed to restore wildcard route: %v\n", err)
		}
	}

	return nil
}

// getAndRemoveWildcardRoute 获取并删除通配符 route
// 返回被删除的 route 配置，以便后续重新添加
func (c *Client) getAndRemoveWildcardRoute() (*Route, error) {
	wildcardID := fmt.Sprintf("wildcard-%s", c.config.CoreDomain)

	// 先获取通配符 route 的配置
	url := fmt.Sprintf("%s/id/%s", c.config.AdminAPI, wildcardID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// 通配符 route 不存在，返回 nil
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get wildcard route (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var route Route
	if err := json.Unmarshal(body, &route); err != nil {
		return nil, err
	}

	// 删除通配符 route
	deleteURL := fmt.Sprintf("%s/id/%s", c.config.AdminAPI, wildcardID)
	deleteReq, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
	if err != nil {
		return nil, err
	}

	deleteResp, err := c.httpClient.Do(deleteReq)
	if err != nil {
		return nil, err
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("failed to delete wildcard route (status %d)", deleteResp.StatusCode)
	}

	return &route, nil
}

// addRoute 添加一个 route
func (c *Client) addRoute(route Route) error {
	body, err := json.Marshal(route)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/config/apps/http/servers/main/routes", c.config.AdminAPI)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add route (status %d)", resp.StatusCode)
	}

	return nil
}

// GetCertificateStatus 查询域名的 SSL 证书状态
// domain: 要查询的域名
func (c *Client) GetCertificateStatus(domain string) (*CertificateInfo, error) {
	url := fmt.Sprintf("%s/config/apps/tls/certificates", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get certificates (status %d): %s", resp.StatusCode, string(respBody))
	}

	// 解析证书列表
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Caddy 的证书 API 返回格式可能是数组或对象，需要根据实际情况解析
	var certsData interface{}
	if err := json.Unmarshal(body, &certsData); err != nil {
		return nil, fmt.Errorf("failed to parse certificates: %w", err)
	}

	// 查找指定域名的证书
	certInfo := &CertificateInfo{
		Domain: domain,
		Status: "not_found",
	}

	// 这里需要根据 Caddy 实际的 API 响应格式进行解析
	// 下面是一个示例实现，可能需要根据实际情况调整
	if certs, ok := certsData.(map[string]interface{}); ok {
		for key, cert := range certs {
			if certMap, ok := cert.(map[string]interface{}); ok {
				// 检查证书是否匹配域名
				if subjects, ok := certMap["subjects"].([]interface{}); ok {
					for _, subject := range subjects {
						if subjectStr, ok := subject.(string); ok && subjectStr == domain {
							certInfo.Status = "issued"

							// 解析证书时间
							if notBefore, ok := certMap["not_before"].(string); ok {
								if t, err := time.Parse(time.RFC3339, notBefore); err == nil {
									certInfo.NotBefore = t
								}
							}
							if notAfter, ok := certMap["not_after"].(string); ok {
								if t, err := time.Parse(time.RFC3339, notAfter); err == nil {
									certInfo.NotAfter = t
								}
							}
							if issuer, ok := certMap["issuer"].(string); ok {
								certInfo.Issuer = issuer
							}

							return certInfo, nil
						}
					}
				}
			}
			_ = key // 避免未使用变量警告
		}
	}

	// 如果在已签发的证书中没有找到，可能正在申请中
	// 检查 ACME 订单状态
	ordersURL := fmt.Sprintf("%s/config/apps/tls/automation/policies/0/subjects", c.config.AdminAPI)
	orderReq, err := http.NewRequest(http.MethodGet, ordersURL, nil)
	if err == nil {
		orderResp, err := c.httpClient.Do(orderReq)
		if err == nil {
			defer orderResp.Body.Close()
			if orderResp.StatusCode == http.StatusOK {
				orderBody, _ := io.ReadAll(orderResp.Body)
				if bytes.Contains(orderBody, []byte(domain)) {
					certInfo.Status = "pending"
					return certInfo, nil
				}
			}
		}
	}

	return certInfo, nil
}

// RemoveStaticSite 移除自定义域名的静态站点
func (c *Client) RemoveStaticSite(domain string) error {
	routeID := fmt.Sprintf("site-%s", sanitizeDomainForID(domain))
	url := fmt.Sprintf("%s/id/%s", c.config.AdminAPI, routeID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove route (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Ping 检查 Caddy Admin API 连接
func (c *Client) Ping() error {
	url := fmt.Sprintf("%s/config/", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Caddy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Caddy returned status: %d", resp.StatusCode)
	}

	return nil
}

// Stop 停止 Caddy 服务器
func (c *Client) Stop() error {
	// 先尝试从 PID 文件读取
	pid, err := c.GetPID()
	if err != nil {
		// 如果没有 PID 文件，尝试使用 cmd
		if c.cmd == nil || c.cmd.Process == nil {
			return fmt.Errorf("Caddy server is not running (no PID file found)")
		}
		pid = c.cmd.Process.Pid
	}

	// 查找进程
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find Caddy process (PID: %d): %w", pid, err)
	}

	// 发送 SIGTERM 信号，优雅关闭
	if err := process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to stop Caddy (PID: %d): %w", pid, err)
	}

	// 等待进程结束
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// 如果 10 秒后还没结束，强制杀死
			if err := process.Kill(); err != nil {
				return fmt.Errorf("failed to kill Caddy (PID: %d): %w", pid, err)
			}
			// 删除 PID 文件
			os.Remove(c.config.PidFile)
			return nil
		case <-ticker.C:
			// 检查进程是否还存在
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// 进程已经结束
				os.Remove(c.config.PidFile)
				return nil
			}
		}
	}
}

// ExportConfig 导出当前 Caddy 的配置到文件
func (c *Client) ExportConfig(outputPath string) error {
	// 从 Admin API 获取当前配置
	url := fmt.Sprintf("%s/config/", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get config (status %d)", resp.StatusCode)
	}

	// 读取配置
	configData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// 格式化 JSON
	var config interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	formattedConfig, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, formattedConfig, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig 获取当前 Caddy 配置
func (c *Client) GetConfig() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/config/", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get config (status %d): %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// sanitizeDomainForID 将域名转换为合法的 ID
// 例如: example.com -> example-com
func sanitizeDomainForID(domain string) string {
	result := ""
	for _, ch := range domain {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			result += string(ch)
		} else {
			result += "-"
		}
	}
	return result
}

// ==================== TLS 策略管理方法 ====================

// AddTLSPolicy 添加 TLS 证书策略
func (c *Client) AddTLSPolicy(policy AutomationPolicy) error {
	url := fmt.Sprintf("%s/config/apps/tls/automation/policies", c.config.AdminAPI)

	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add TLS policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add TLS policy (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetTLSPolicies 获取所有 TLS 证书策略
func (c *Client) GetTLSPolicies() ([]AutomationPolicy, error) {
	url := fmt.Sprintf("%s/config/apps/tls/automation/policies", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// TLS 配置可能不存在
		return []AutomationPolicy{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get TLS policies (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var policies []AutomationPolicy
	if err := json.Unmarshal(body, &policies); err != nil {
		return nil, fmt.Errorf("failed to parse policies: %w", err)
	}

	return policies, nil
}

// RemoveTLSPolicy 移除 TLS 证书策略
func (c *Client) RemoveTLSPolicy(policyID string) error {
	url := fmt.Sprintf("%s/id/%s", c.config.AdminAPI, policyID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove TLS policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove TLS policy (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ==================== 域名检查方法 ====================

// CheckDomainReadiness 检查域名是否就绪（供 API 层调用）
func (c *Client) CheckDomainReadiness(domain string) (*DomainCheckResult, error) {
	if c.checker == nil {
		// 如果没有配置 ServerIP，创建临时检查器
		checker := NewDomainChecker("")
		return checker.CheckAll(domain), nil
	}
	return c.checker.CheckAll(domain), nil
}

// ==================== 自定义域名管理方法 ====================

// AddCustomDomain 添加用户自定义域名（单域名模式）
// 每个 License 只能有一个自定义域名，因此每次只处理单个域名
// skipCheck: 是否跳过域名预检测（仅开发环境使用）
//
// 注意：此方法仅用于用户自定义域名（如 hello.com），不适用于平台 subdomain。
// 平台 subdomain（如 user123.mdfriday.com）应使用 AddStaticSite，因为它们使用 Wildcard 证书。
func (c *Client) AddCustomDomain(domain, sitePath string, skipCheck bool) error {
	// 0. 检查是否为平台域名（使用 Wildcard 证书）
	// 平台 subdomain 应使用 AddStaticSite，不需要单独的 TLS policy
	if IsPlatformDomain(domain, c.config.CoreDomain) {
		// 如果是平台域名，只添加 route，不添加 TLS policy
		return c.AddStaticSite(domain, sitePath)
	}

	// 1. 域名预检测（如果启用）
	if !skipCheck && c.checker != nil {
		result := c.checker.CheckAll(domain)
		if !result.Ready {
			return fmt.Errorf("domain not ready: %s", result.Error)
		}
	}

	// 2. 添加 HTTP route
	if err := c.AddStaticSite(domain, sitePath); err != nil {
		return fmt.Errorf("failed to add static site: %w", err)
	}

	// 3. 创建并添加 TLS policy（单域名，使用 HTTP-01 challenge）
	policy := NewSingleDomainHTTP01Policy(domain)

	if err := c.AddTLSPolicy(policy); err != nil {
		// 回滚：移除已添加的 route
		c.RemoveStaticSite(domain)
		return fmt.Errorf("failed to add TLS policy: %w", err)
	}

	return nil
}

// RemoveCustomDomain 移除用户自定义域名
// 注意：平台 subdomain 应使用 RemoveStaticSite
func (c *Client) RemoveCustomDomain(domain string) error {
	// 检查是否为平台域名
	if IsPlatformDomain(domain, c.config.CoreDomain) {
		// 平台域名只需移除 route，没有单独的 TLS policy
		return c.RemoveStaticSite(domain)
	}

	var lastErr error

	// 1. 移除 HTTP route
	if err := c.RemoveStaticSite(domain); err != nil {
		// 记录错误但继续尝试移除 TLS policy
		lastErr = fmt.Errorf("failed to remove static site: %w", err)
	}

	// 2. 移除 TLS policy（仅非平台域名需要）
	policyID := "custom-" + sanitizeDomainForID(domain)
	if err := c.RemoveTLSPolicy(policyID); err != nil {
		if lastErr != nil {
			return fmt.Errorf("multiple errors: %v; %w", lastErr, err)
		}
		lastErr = fmt.Errorf("failed to remove TLS policy: %w", err)
	}

	return lastErr
}

// ==================== 平台域名 Wildcard 证书 ====================

// SetupPlatformWildcard 设置平台域名的 Wildcard 证书
// 使用 DNS-01 challenge
// 注意：凭据直接在配置中传递
func (c *Client) SetupPlatformWildcard() error {
	if c.config.DNSPodToken == "" {
		return fmt.Errorf("DNSPodToken is required for wildcard certificate")
	}
	if c.config.CoreDomain == "" || c.config.CoreDomain == "localhost" || c.config.CoreDomain == "127.0.0.1" {
		return fmt.Errorf("valid CoreDomain is required for wildcard certificate")
	}

	secretID, secretKey := c.parseDNSPodToken()
	if secretID == "" || secretKey == "" {
		return fmt.Errorf("invalid DNSPodToken format, expected 'SecretId,SecretKey'")
	}

	policy := NewWildcardDNS01Policy(c.config.CoreDomain, "tencentcloud", secretID, secretKey)
	return c.AddTLSPolicy(policy)
}

// ==================== 配置生成辅助方法 ====================

// GenerateTLSConfig 生成 TLS 配置（用于启动时）
// 凭据直接在配置中传递
func (c *Client) GenerateTLSConfig() *TLSConfig {
	// 如果没有配置 DNSPodToken 或是开发环境，不生成 TLS 配置
	isDev := c.config.CoreDomain == "localhost" || c.config.CoreDomain == "127.0.0.1"
	if c.config.DNSPodToken == "" || isDev {
		return nil
	}

	secretID, secretKey := c.parseDNSPodToken()
	if secretID == "" || secretKey == "" {
		return nil
	}

	// 生成平台域名的 Wildcard 策略
	wildcardPolicy := NewWildcardDNS01Policy(c.config.CoreDomain, "tencentcloud", secretID, secretKey)

	return &TLSConfig{
		Automation: &AutomationConfig{
			Policies: []AutomationPolicy{wildcardPolicy},
		},
	}
}

// parseDNSPodToken 解析 DNSPodToken 字符串
// DNSPodToken 格式为 "SecretId,SecretKey"（逗号分隔）
// 返回 secretID 和 secretKey，如果格式不正确则返回空字符串
func (c *Client) parseDNSPodToken() (secretID, secretKey string) {
	if c.config.DNSPodToken == "" {
		return "", ""
	}
	parts := strings.SplitN(c.config.DNSPodToken, ",", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
