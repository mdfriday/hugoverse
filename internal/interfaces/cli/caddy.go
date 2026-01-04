package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
)

type caddyCmd struct {
	parent *flag.FlagSet
	cmd    *flag.FlagSet
}

// NewCaddyCmd 创建 Caddy 管理命令
func NewCaddyCmd(parent *flag.FlagSet) (*caddyCmd, error) {
	nCmd := &caddyCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("caddy", flag.ExitOnError)
	nCmd.cmd.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("  hugov caddy [subcommand]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  start    Start Caddy server in background")
		fmt.Println("  stop     Stop Caddy server")
		fmt.Println("  status   Check Caddy server status")
		fmt.Println("  add      Add a static site")
		fmt.Println("  remove   Remove a static site")
		fmt.Println("  cert     Check SSL certificate status")
		fmt.Println("  export   Export current Caddy configuration to file")
		fmt.Println("\nExamples:")
		fmt.Println("  # Start in development mode (localhost, HTTP only)")
		fmt.Println("  hugov caddy start")
		fmt.Println("  hugov caddy start -domain localhost -backend 127.0.0.1:1314 -couchdb 127.0.0.1:5984")
		fmt.Println("")
		fmt.Println("  # Start in production mode (HTTPS auto-enabled)")
		fmt.Println("  hugov caddy start -domain mdfriday.site")
		fmt.Println("")
		fmt.Println("  # Add a static site")
		fmt.Println("  hugov caddy add -domain example.com -path /web/sites/example-com")
		fmt.Println("")
		fmt.Println("  # Check certificate status")
		fmt.Println("  hugov caddy cert -domain example.com")
		fmt.Println("")
		fmt.Println("  # Export configuration")
		fmt.Println("  hugov caddy export -output /tmp/caddy-backup.json")
		fmt.Println("")
		fmt.Println("  # Stop server")
		fmt.Println("  hugov caddy stop")
		fmt.Println("\nDefault Routes:")
		fmt.Println("  - {domain}       → Default backend (Hugo/App server)")
		fmt.Println("  - cdb.{domain}   → CouchDB service")
	}

	err := nCmd.cmd.Parse(parent.Args()[1:])
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (c *caddyCmd) Usage() {
	c.cmd.Usage()
}

func (c *caddyCmd) Run() error {
	args := c.cmd.Args()
	if len(args) == 0 {
		c.Usage()
		return fmt.Errorf("please specify a subcommand")
	}

	subCommand := args[0]

	switch subCommand {
	case "start":
		return c.runStart(args[1:])
	case "stop":
		return c.runStop(args[1:])
	case "status":
		return c.runStatus(args[1:])
	case "add":
		return c.runAdd(args[1:])
	case "remove":
		return c.runRemove(args[1:])
	case "cert":
		return c.runCert(args[1:])
	case "export":
		return c.runExport(args[1:])
	default:
		c.Usage()
		return fmt.Errorf("unknown subcommand: %s", subCommand)
	}
}

// runStart 启动 Caddy 服务器（后台运行）
func (c *caddyCmd) runStart(args []string) error {
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	adminAPI := startCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	backend := startCmd.String("backend", "127.0.0.1:1314", "Default backend service")
	couchdb := startCmd.String("couchdb", "127.0.0.1:5984", "CouchDB backend service")
	domain := startCmd.String("domain", "localhost", "Core domain (use localhost for dev)")
	configPath := startCmd.String("config", "/tmp/caddy-config.json", "Caddy config file path")
	pidFile := startCmd.String("pid", "/tmp/caddy.pid", "PID file path")
	logFile := startCmd.String("log", "/tmp/caddy.log", "Log file path")

	if err := startCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("🚀 Starting Caddy server in background...")
	fmt.Printf("   Admin API: %s\n", *adminAPI)
	fmt.Printf("   Backend: %s\n", *backend)
	fmt.Printf("   CouchDB: %s\n", *couchdb)
	fmt.Printf("   Domain: %s\n", *domain)
	fmt.Printf("   Config: %s\n", *configPath)
	fmt.Printf("   PID File: %s\n", *pidFile)
	fmt.Printf("   Log File: %s\n", *logFile)

	// 判断是否为开发环境
	isDev := *domain == "localhost" || *domain == "127.0.0.1"
	if isDev {
		fmt.Println("   Mode: Development (HTTP only, no sudo required)")
	} else {
		fmt.Println("   Mode: Production (HTTPS auto-enabled)")
	}

	// 创建 Caddy 客户端配置
	config := &caddy.Config{
		AdminAPI:       *adminAPI,
		BinaryPath:     "caddy",
		DefaultBackend: *backend,
		CouchDBBackend: *couchdb,
		CoreDomain:     *domain,
		ConfigPath:     *configPath,
		PidFile:        *pidFile,
		LogFile:        *logFile,
	}

	client := caddy.NewClient(config)

	// 检查是否已经在运行
	if client.IsRunning() {
		fmt.Println("⚠️  Caddy is already running")
		pid, _ := client.GetPID()
		fmt.Printf("   PID: %d\n", pid)
		fmt.Printf("   Use 'hugov caddy status' to check status\n")
		fmt.Printf("   Use 'hugov caddy stop' to stop\n")
		return nil
	}

	// 启动服务器（后台）
	if err := client.StartServerBackground(); err != nil {
		return fmt.Errorf("failed to start Caddy: %w", err)
	}

	pid, _ := client.GetPID()
	fmt.Println("✅ Caddy server started successfully in background")
	fmt.Printf("   PID: %d\n", pid)
	
	// 根据环境显示不同的访问信息
	if isDev {
		fmt.Println("\n📡 Access URLs:")
		fmt.Printf("   Core Service:  http://%s:8080\n", *domain)
		fmt.Printf("   CouchDB:       http://cdb.%s:8080\n", *domain)
		fmt.Println("\n   Note: Development mode uses port 8080 (no sudo required)")
	} else {
		fmt.Println("\n📡 Access URLs:")
		fmt.Printf("   Core Service (HTTP):   http://%s\n", *domain)
		fmt.Printf("   Core Service (HTTPS):  https://%s (certificate will be auto-issued)\n", *domain)
		fmt.Printf("   CouchDB (HTTP):        http://cdb.%s\n", *domain)
		fmt.Printf("   CouchDB (HTTPS):       https://cdb.%s (certificate will be auto-issued)\n", *domain)
		fmt.Println("\n   Note: SSL certificates will be automatically issued by Let's Encrypt")
	}
	
	fmt.Println("\n💡 Tips:")
	fmt.Println("   - Use 'hugov caddy status' to check server status")
	fmt.Println("   - Use 'hugov caddy add' to add static sites")
	fmt.Println("   - Use 'hugov caddy stop' to stop the server")
	fmt.Println("   - Use 'hugov caddy export' to export current config")
	fmt.Printf("   - Logs: tail -f %s\n", *logFile)

	return nil
}

// runStop 停止 Caddy 服务器
func (c *caddyCmd) runStop(args []string) error {
	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	pidFile := stopCmd.String("pid", "/tmp/caddy.pid", "PID file path")
	adminAPI := stopCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")

	if err := stopCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("🛑 Stopping Caddy server...")

	// 创建 Caddy 客户端
	config := &caddy.Config{
		AdminAPI:   *adminAPI,
		BinaryPath: "caddy",
		PidFile:    *pidFile,
	}

	client := caddy.NewClient(config)

	// 检查是否在运行
	if !client.IsRunning() {
		fmt.Println("⚠️  Caddy is not running")
		return nil
	}

	pid, _ := client.GetPID()
	fmt.Printf("   Stopping Caddy (PID: %d)...\n", pid)

	// 停止服务器
	if err := client.Stop(); err != nil {
		return fmt.Errorf("failed to stop Caddy: %w", err)
	}

	fmt.Println("✅ Caddy server stopped successfully")
	return nil
}

// runStatus 检查 Caddy 状态
func (c *caddyCmd) runStatus(args []string) error {
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	adminAPI := statusCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")

	if err := statusCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("🔍 Checking Caddy server status...")

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 检查连接
	if err := client.Ping(); err != nil {
		fmt.Println("❌ Caddy server is not running or not responding")
		fmt.Printf("   Error: %v\n", err)
		return nil
	}

	fmt.Println("✅ Caddy server is running")

	// 获取配置信息
	config2, err := client.GetConfig()
	if err != nil {
		fmt.Printf("⚠️  Failed to get config: %v\n", err)
		return nil
	}

	// 显示路由数量
	if apps, ok := config2["apps"].(map[string]interface{}); ok {
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				if mainServer, ok := servers["main"].(map[string]interface{}); ok {
					if routes, ok := mainServer["routes"].([]interface{}); ok {
						fmt.Printf("   Total routes: %d\n", len(routes))
						
						// 显示所有域名
						fmt.Println("\n📋 Configured domains:")
						for i, route := range routes {
							if routeMap, ok := route.(map[string]interface{}); ok {
								if matches, ok := routeMap["match"].([]interface{}); ok {
									for _, match := range matches {
										if matchMap, ok := match.(map[string]interface{}); ok {
											if hosts, ok := matchMap["host"].([]interface{}); ok {
												for _, host := range hosts {
													fmt.Printf("   %d. %s\n", i+1, host)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("\n   Admin API: %s\n", *adminAPI)
	fmt.Println("   View full config: curl http://127.0.0.1:2019/config/ | jq")

	return nil
}

// runAdd 添加静态站点
func (c *caddyCmd) runAdd(args []string) error {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	adminAPI := addCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := addCmd.String("domain", "", "Domain name (required)")
	path := addCmd.String("path", "", "Static site path (required)")

	if err := addCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy add -domain example.com -path /path/to/site")
	}
	if *path == "" {
		return fmt.Errorf("path is required. Use: hugov caddy add -domain example.com -path /path/to/site")
	}

	fmt.Printf("📁 Adding static site: %s -> %s\n", *domain, *path)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 添加站点
	if err := client.AddStaticSite(*domain, *path); err != nil {
		return fmt.Errorf("failed to add site: %w", err)
	}

	fmt.Println("✅ Static site added successfully")
	fmt.Printf("   Domain: %s\n", *domain)
	fmt.Printf("   Path: %s\n", *path)
	
	if *domain != "localhost" && *domain != "127.0.0.1" {
		fmt.Println("\n⏳ SSL certificate will be auto-issued by Let's Encrypt")
		fmt.Println("   Check status with: hugov caddy cert -domain " + *domain)
	}

	return nil
}

// runRemove 移除静态站点
func (c *caddyCmd) runRemove(args []string) error {
	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	adminAPI := removeCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := removeCmd.String("domain", "", "Domain name (required)")

	if err := removeCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy remove -domain example.com")
	}

	fmt.Printf("🗑️  Removing static site: %s\n", *domain)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 移除站点
	if err := client.RemoveStaticSite(*domain); err != nil {
		return fmt.Errorf("failed to remove site: %w", err)
	}

	fmt.Println("✅ Static site removed successfully")

	return nil
}

// runCert 查询证书状态
func (c *caddyCmd) runCert(args []string) error {
	certCmd := flag.NewFlagSet("cert", flag.ExitOnError)
	adminAPI := certCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := certCmd.String("domain", "", "Domain name (required)")

	if err := certCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy cert -domain example.com")
	}

	fmt.Printf("📜 Checking SSL certificate for: %s\n", *domain)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 查询证书
	certInfo, err := client.GetCertificateStatus(*domain)
	if err != nil {
		return fmt.Errorf("failed to get certificate status: %w", err)
	}

	// 显示状态
	statusEmoji := "❓"
	statusText := certInfo.Status
	
	switch certInfo.Status {
	case "issued":
		statusEmoji = "✅"
		statusText = "Issued"
	case "pending":
		statusEmoji = "⏳"
		statusText = "Pending (being issued)"
	case "not_found":
		statusEmoji = "🔍"
		statusText = "Not found (not yet requested)"
	case "failed":
		statusEmoji = "❌"
		statusText = "Failed"
	}

	fmt.Printf("\n%s Status: %s\n", statusEmoji, statusText)

	if certInfo.Status == "issued" {
		fmt.Println("\n📋 Certificate details:")
		fmt.Printf("   Domain: %s\n", certInfo.Domain)
		fmt.Printf("   Issuer: %s\n", certInfo.Issuer)
		fmt.Printf("   Valid from: %s\n", certInfo.NotBefore.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Valid until: %s\n", certInfo.NotAfter.Format("2006-01-02 15:04:05"))
		
		// 计算剩余天数
		daysLeft := int(time.Until(certInfo.NotAfter).Hours() / 24)
		fmt.Printf("   Days remaining: %d\n", daysLeft)
		
		if daysLeft < 30 {
			fmt.Println("   ⚠️  Certificate will expire soon!")
		}
	} else if certInfo.Status == "pending" {
		fmt.Println("\n💡 Certificate is being issued. This usually takes a few seconds.")
		fmt.Println("   Check again in a moment.")
	} else if certInfo.Status == "not_found" {
		fmt.Println("\n💡 No certificate found. Possible reasons:")
		fmt.Println("   - Domain was just added (certificate is being requested)")
		fmt.Println("   - Domain is not accessible from the internet")
		fmt.Println("   - DNS not configured correctly")
	} else if certInfo.Error != "" {
		fmt.Printf("\n❌ Error: %s\n", certInfo.Error)
	}

	return nil
}

// runExport 导出当前 Caddy 配置
func (c *caddyCmd) runExport(args []string) error {
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	adminAPI := exportCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	output := exportCmd.String("output", "/tmp/caddy-config-export.json", "Output file path")

	if err := exportCmd.Parse(args); err != nil {
		return err
	}

	fmt.Printf("📤 Exporting Caddy configuration...\n")
	fmt.Printf("   Admin API: %s\n", *adminAPI)
	fmt.Printf("   Output: %s\n", *output)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 导出配置
	if err := client.ExportConfig(*output); err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	fmt.Println("✅ Configuration exported successfully")
	fmt.Println("\n💡 Tips:")
	fmt.Println("   - Use this file to restore configuration later")
	fmt.Printf("   - Start with config: hugov caddy start -config %s\n", *output)
	fmt.Printf("   - Or use: caddy run --config %s\n", *output)

	return nil
}
