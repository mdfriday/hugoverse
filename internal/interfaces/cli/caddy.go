package cli

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
		fmt.Println("  start    Start Caddy server")
		fmt.Println("  stop     Stop Caddy server")
		fmt.Println("  status   Check Caddy server status")
		fmt.Println("  add      Add a static site")
		fmt.Println("  remove   Remove a static site")
		fmt.Println("  cert     Check SSL certificate status")
		fmt.Println("\nExamples:")
		fmt.Println("  hugov caddy start")
		fmt.Println("  hugov caddy add -domain example.com -path /web/sites/example-com")
		fmt.Println("  hugov caddy cert -domain example.com")
		fmt.Println("  hugov caddy stop")
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
	default:
		c.Usage()
		return fmt.Errorf("unknown subcommand: %s", subCommand)
	}
}

// runStart 启动 Caddy 服务器
func (c *caddyCmd) runStart(args []string) error {
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	adminAPI := startCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	backend := startCmd.String("backend", "127.0.0.1:1314", "Default backend service")
	domain := startCmd.String("domain", "localhost", "Core domain (use localhost for dev)")
	configPath := startCmd.String("config", "", "Caddy config file path (optional)")

	if err := startCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("🚀 Starting Caddy server...")
	fmt.Printf("   Admin API: %s\n", *adminAPI)
	fmt.Printf("   Backend: %s\n", *backend)
	fmt.Printf("   Domain: %s\n", *domain)

	// 判断是否为开发环境
	isDev := *domain == "localhost" || *domain == "127.0.0.1"
	if isDev {
		fmt.Println("   Mode: Development (HTTP only, no sudo required)")
	} else {
		fmt.Println("   Mode: Production (HTTPS auto-enabled)")
	}

	// 创建 Caddy 客户端配置（开发环境使用 localhost）
	config := &caddy.Config{
		AdminAPI:       *adminAPI,
		BinaryPath:     "caddy",
		DefaultBackend: *backend,
		CoreDomain:     *domain,
		ConfigPath:     *configPath,
	}

	client := caddy.NewClient(config)

	// 启动服务器
	if err := client.StartServer(); err != nil {
		return fmt.Errorf("failed to start Caddy: %w", err)
	}

	fmt.Println("✅ Caddy server started successfully")
	
	// 根据环境显示不同的访问信息
	if isDev {
		fmt.Printf("   Access via: http://%s:8080\n", *domain)
		fmt.Println("   Note: Development mode uses port 8080 (no sudo required)")
	} else {
		fmt.Printf("   HTTP: http://%s\n", *domain)
		fmt.Printf("   HTTPS: https://%s (certificate will be auto-issued)\n", *domain)
	}
	
	fmt.Println("\n💡 Tips:")
	fmt.Println("   - Use 'hugov caddy status' to check server status")
	fmt.Println("   - Use 'hugov caddy add' to add static sites")
	fmt.Println("   - Press Ctrl+C to stop the server")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\n🛑 Stopping Caddy server...")
	if err := client.Stop(); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	} else {
		fmt.Println("✅ Caddy server stopped")
	}

	return nil
}

// runStop 停止 Caddy 服务器
func (c *caddyCmd) runStop(args []string) error {
	fmt.Println("🛑 Stopping Caddy server...")
	fmt.Println("💡 Note: Use Ctrl+C in the running terminal to stop Caddy")
	fmt.Println("   Or use: pkill caddy")
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

