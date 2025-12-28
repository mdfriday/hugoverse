package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// Config Caddy 配置
type Config struct {
	AdminAPI       string `json:"admin_api"`        // Caddy Admin API 地址 (如 http://127.0.0.1:2019)
	ConfigPath     string `json:"config_path"`      // Caddy 配置文件路径
	BinaryPath     string `json:"binary_path"`      // Caddy 二进制文件路径
	DefaultBackend string `json:"default_backend"`  // 默认后端服务地址 (如 127.0.0.1:1314)
	CoreDomain     string `json:"core_domain"`      // 核心域名 (如 mdfriday.site)
}

// Client Caddy HTTP 客户端
type Client struct {
	config     *Config
	httpClient *http.Client
	cmd        *exec.Cmd // 用于管理 Caddy 进程
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
	if config.CoreDomain == "" {
		config.CoreDomain = "mdfriday.site"
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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
}

// HTTPConfig HTTP App 配置
type HTTPConfig struct {
	Servers map[string]*ServerConfig `json:"servers"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Listen    []string `json:"listen"`
	Routes    []Route  `json:"routes"`
	AutoHTTPS *AutoHTTPSConfig `json:"automatic_https,omitempty"` // 自动 HTTPS 配置
}

// AutoHTTPSConfig 自动 HTTPS 配置
type AutoHTTPSConfig struct {
	Disable bool `json:"disable,omitempty"` // 禁用自动 HTTPS
}

// Route 路由配置
type Route struct {
	ID     string          `json:"@id,omitempty"`
	Match  []MatchHost     `json:"match"`
	Handle []HandleConfig  `json:"handle"`
}

// MatchHost 域名匹配
type MatchHost struct {
	Host []string `json:"host"`
}

// HandleConfig 处理器配置
type HandleConfig struct {
	Handler   string      `json:"handler"`
	Upstreams []Upstream  `json:"upstreams,omitempty"` // for reverse_proxy
	Root      string      `json:"root,omitempty"`      // for file_server
}

// Upstream 上游服务器
type Upstream struct {
	Dial string `json:"dial"`
}

// CertificateInfo SSL 证书信息
type CertificateInfo struct {
	Domain     string    `json:"domain"`
	Status     string    `json:"status"`      // "issued", "pending", "failed"
	NotBefore  time.Time `json:"not_before"`
	NotAfter   time.Time `json:"not_after"`
	Issuer     string    `json:"issuer"`
	Error      string    `json:"error,omitempty"`
}

// StartServer 启动 Caddy 服务器
// 使用默认配置启动，包含核心域名的反向代理
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

// AddStaticSite 动态添加自定义域名的静态站点
// domain: 自定义域名 (如 example.com)
// sitePath: 静态站点文件路径 (如 /web/sites/example-com)
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

	body, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	url := fmt.Sprintf("%s/config/apps/http/servers/main/routes", c.config.AdminAPI)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add route (status %d): %s", resp.StatusCode, string(respBody))
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
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("Caddy server is not running")
	}

	// 发送 SIGTERM 信号，优雅关闭
	if err := c.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to stop Caddy: %w", err)
	}

	// 等待进程结束
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		// 如果 10 秒后还没结束，强制杀死
		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill Caddy: %w", err)
		}
		return nil
	}
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

