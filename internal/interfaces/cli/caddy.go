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
		fmt.Println("  add      Add a static site (subdomain)")
		fmt.Println("  remove   Remove a static site (subdomain)")
		fmt.Println("  domain   Custom domain management (with TLS)")
		fmt.Println("  tls      TLS policy management")
		fmt.Println("  cert     Check SSL certificate status")
		fmt.Println("  export   Export current Caddy configuration to file")
		fmt.Println("\nExamples:")
		fmt.Println("  # Start in development mode (localhost, HTTP only)")
		fmt.Println("  hugov caddy start")
		fmt.Println("  hugov caddy start -domain localhost -backend 127.0.0.1:1314 -couchdb 127.0.0.1:5984")
		fmt.Println("")
		fmt.Println("  # Start in production mode (HTTPS with Wildcard certificate)")
		fmt.Println("  hugov caddy start -domain mdfriday.site -dnspod-token $DNSPOD_API_TOKEN -server-ip 1.2.3.4")
		fmt.Println("")
		fmt.Println("  # Add a subdomain static site")
		fmt.Println("  hugov caddy add -domain user123.mdfriday.site -path /web/sites/user123")
		fmt.Println("")
		fmt.Println("  # Custom domain management (with TLS)")
		fmt.Println("  hugov caddy domain check -domain hello.com -server-ip 1.2.3.4")
		fmt.Println("  hugov caddy domain add -domain hello.com -path /web/sites/hello-com")
		fmt.Println("  hugov caddy domain remove -domain hello.com")
		fmt.Println("")
		fmt.Println("  # TLS policy management")
		fmt.Println("  hugov caddy tls policies")
		fmt.Println("  hugov caddy tls add-policy -domain hello.com")
		fmt.Println("  hugov caddy tls remove-policy -id custom-hello-com")
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
	case "domain":
		return c.runDomain(args[1:])
	case "tls":
		return c.runTLS(args[1:])
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
	dnspodToken := startCmd.String("dnspod-token", "", "腾讯云 DNS API token (格式: SecretId,SecretKey) for wildcard certificate")
	serverIP := startCmd.String("server-ip", "", "Server public IP for domain verification")

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
	if *serverIP != "" {
		fmt.Printf("   Server IP: %s\n", *serverIP)
	}
	if *dnspodToken != "" {
		fmt.Println("   DNSPod Token: ***configured***")
	}

	// 判断是否为开发环境
	isDev := *domain == "localhost" || *domain == "127.0.0.1"
	if isDev {
		fmt.Println("   Mode: Development (HTTP only, no sudo required)")
	} else {
		if *dnspodToken != "" {
			fmt.Println("   Mode: Production (HTTPS with Wildcard certificate)")
		} else {
			fmt.Println("   Mode: Production (HTTPS auto-enabled)")
		}
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
		DNSPodToken:    *dnspodToken,
		ServerIP:       *serverIP,
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
	coreDomain := addCmd.String("core-domain", "mdfriday.com", "Core domain for wildcard routing")
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
		AdminAPI:   *adminAPI,
		CoreDomain: *coreDomain,
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

// ==================== Domain 子命令 ====================

// runDomain 处理 domain 子命令
func (c *caddyCmd) runDomain(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: hugov caddy domain [subcommand]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  check   Check domain readiness (DNS + HTTP)")
		fmt.Println("  add     Add custom domain with TLS")
		fmt.Println("  remove  Remove custom domain")
		fmt.Println("  list    List all custom domains")
		return fmt.Errorf("please specify a subcommand")
	}

	subCommand := args[0]
	switch subCommand {
	case "check":
		return c.runDomainCheck(args[1:])
	case "add":
		return c.runDomainAdd(args[1:])
	case "remove":
		return c.runDomainRemove(args[1:])
	case "list":
		return c.runDomainList(args[1:])
	default:
		return fmt.Errorf("unknown domain subcommand: %s", subCommand)
	}
}

// runDomainCheck 检查域名就绪状态
func (c *caddyCmd) runDomainCheck(args []string) error {
	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	domain := checkCmd.String("domain", "", "Domain name to check (required)")
	serverIP := checkCmd.String("server-ip", "", "Server public IP for verification")

	if err := checkCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy domain check -domain hello.com")
	}

	fmt.Printf("🔍 Checking domain readiness: %s\n", *domain)
	if *serverIP != "" {
		fmt.Printf("   Expected IP: %s\n", *serverIP)
	}

	// 创建域名检查器
	checker := caddy.NewDomainChecker(*serverIP)
	result := checker.CheckAll(*domain)

	// 显示结果
	fmt.Println("\n📋 Check Results:")
	
	// DNS 检查
	if result.DNSValid {
		fmt.Printf("   ✅ DNS Valid: true\n")
		fmt.Printf("      Resolved IPs: %v\n", result.ResolvedIPs)
	} else {
		fmt.Printf("   ❌ DNS Valid: false\n")
		if len(result.ResolvedIPs) > 0 {
			fmt.Printf("      Resolved IPs: %v\n", result.ResolvedIPs)
		}
	}

	// HTTP 检查
	if result.HTTPReachable {
		fmt.Printf("   ✅ HTTP Reachable: true\n")
	} else {
		fmt.Printf("   ❌ HTTP Reachable: false\n")
	}

	// 总体结果
	if result.Ready {
		fmt.Println("\n✅ Domain is ready for HTTPS certificate issuance")
		fmt.Println("   You can now add this domain: hugov caddy domain add -domain " + *domain + " -path /path/to/site")
	} else {
		fmt.Printf("\n❌ Domain is not ready: %s\n", result.Error)
		fmt.Println("\n💡 Tips:")
		if !result.DNSValid {
			fmt.Println("   1. Configure your domain DNS to point to this server")
			if *serverIP != "" {
				fmt.Printf("      Add an A record: %s -> %s\n", *domain, *serverIP)
			}
			fmt.Println("   2. Wait for DNS propagation (may take a few minutes)")
		}
		if !result.HTTPReachable {
			fmt.Println("   - Ensure the server is accessible on port 80")
			fmt.Println("   - Check firewall settings")
		}
	}

	return nil
}

// runDomainAdd 添加自定义域名
func (c *caddyCmd) runDomainAdd(args []string) error {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	adminAPI := addCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := addCmd.String("domain", "", "Custom domain name (required)")
	path := addCmd.String("path", "", "Static site path (required)")
	serverIP := addCmd.String("server-ip", "", "Server public IP for verification")
	skipCheck := addCmd.Bool("skip-check", false, "Skip domain readiness check (dev only)")

	if err := addCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy domain add -domain hello.com -path /path/to/site")
	}
	if *path == "" {
		return fmt.Errorf("path is required. Use: hugov caddy domain add -domain hello.com -path /path/to/site")
	}

	fmt.Printf("🌐 Adding custom domain: %s -> %s\n", *domain, *path)
	if *skipCheck {
		fmt.Println("   ⚠️  Skipping domain readiness check (development mode)")
	}

	config := &caddy.Config{
		AdminAPI: *adminAPI,
		ServerIP: *serverIP,
	}
	client := caddy.NewClient(config)

	// 添加自定义域名（包含 TLS 策略）
	if err := client.AddCustomDomain(*domain, *path, *skipCheck); err != nil {
		return fmt.Errorf("failed to add custom domain: %w", err)
	}

	fmt.Println("✅ Custom domain added successfully")
	fmt.Printf("   Domain: %s\n", *domain)
	fmt.Printf("   Path: %s\n", *path)
	fmt.Println("\n⏳ SSL certificate will be auto-issued by Let's Encrypt")
	fmt.Println("   This usually takes a few seconds to a minute.")
	fmt.Println("   Check status with: hugov caddy cert -domain " + *domain)

	return nil
}

// runDomainRemove 移除自定义域名
func (c *caddyCmd) runDomainRemove(args []string) error {
	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	adminAPI := removeCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := removeCmd.String("domain", "", "Custom domain name (required)")

	if err := removeCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy domain remove -domain hello.com")
	}

	fmt.Printf("🗑️  Removing custom domain: %s\n", *domain)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	// 移除自定义域名（包含 TLS 策略）
	if err := client.RemoveCustomDomain(*domain); err != nil {
		return fmt.Errorf("failed to remove custom domain: %w", err)
	}

	fmt.Println("✅ Custom domain removed successfully")

	return nil
}

// runDomainList 列出所有自定义域名
func (c *caddyCmd) runDomainList(args []string) error {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	adminAPI := listCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")

	if err := listCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("📋 Custom domains (TLS policies):")

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	policies, err := client.GetTLSPolicies()
	if err != nil {
		return fmt.Errorf("failed to get TLS policies: %w", err)
	}

	if len(policies) == 0 {
		fmt.Println("   No TLS policies configured")
		return nil
	}

	customCount := 0
	for _, policy := range policies {
		// 只显示自定义域名的策略（ID 以 custom- 开头）
		if len(policy.ID) > 7 && policy.ID[:7] == "custom-" {
			customCount++
			fmt.Printf("\n   %d. Policy: %s\n", customCount, policy.ID)
			fmt.Printf("      Domains: %v\n", policy.Subjects)
		}
	}

	if customCount == 0 {
		fmt.Println("   No custom domains configured")
		fmt.Println("\n💡 Add a custom domain:")
		fmt.Println("   hugov caddy domain add -domain hello.com -path /path/to/site")
	} else {
		fmt.Printf("\n   Total: %d custom domain(s)\n", customCount)
	}

	return nil
}

// ==================== TLS 子命令 ====================

// runTLS 处理 tls 子命令
func (c *caddyCmd) runTLS(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: hugov caddy tls [subcommand]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  policies       List all TLS policies")
		fmt.Println("  add-policy     Add a TLS policy")
		fmt.Println("  remove-policy  Remove a TLS policy")
		return fmt.Errorf("please specify a subcommand")
	}

	subCommand := args[0]
	switch subCommand {
	case "policies":
		return c.runTLSPolicies(args[1:])
	case "add-policy":
		return c.runTLSAddPolicy(args[1:])
	case "remove-policy":
		return c.runTLSRemovePolicy(args[1:])
	default:
		return fmt.Errorf("unknown tls subcommand: %s", subCommand)
	}
}

// runTLSPolicies 查看所有 TLS 策略
func (c *caddyCmd) runTLSPolicies(args []string) error {
	policiesCmd := flag.NewFlagSet("policies", flag.ExitOnError)
	adminAPI := policiesCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")

	if err := policiesCmd.Parse(args); err != nil {
		return err
	}

	fmt.Println("📜 TLS Automation Policies:")

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	policies, err := client.GetTLSPolicies()
	if err != nil {
		return fmt.Errorf("failed to get TLS policies: %w", err)
	}

	if len(policies) == 0 {
		fmt.Println("   No TLS policies configured")
		return nil
	}

	for i, policy := range policies {
		fmt.Printf("\n   %d. ID: %s\n", i+1, policy.ID)
		fmt.Printf("      Subjects: %v\n", policy.Subjects)
		if len(policy.Issuers) > 0 {
			issuer := policy.Issuers[0]
			fmt.Printf("      Module: %s\n", issuer.Module)
			if issuer.Challenges != nil {
				if issuer.Challenges.DNS != nil {
					fmt.Printf("      Challenge: DNS-01 (%s)\n", issuer.Challenges.DNS.Provider.Name)
				} else if issuer.Challenges.HTTP != nil {
					fmt.Println("      Challenge: HTTP-01")
				}
			}
		}
	}

	fmt.Printf("\n   Total: %d policy/policies\n", len(policies))

	return nil
}

// runTLSAddPolicy 添加 TLS 策略
func (c *caddyCmd) runTLSAddPolicy(args []string) error {
	addCmd := flag.NewFlagSet("add-policy", flag.ExitOnError)
	adminAPI := addCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	domain := addCmd.String("domain", "", "Domain name (required)")
	challenge := addCmd.String("challenge", "http", "Challenge type: http or dns")

	if err := addCmd.Parse(args); err != nil {
		return err
	}

	if *domain == "" {
		return fmt.Errorf("domain is required. Use: hugov caddy tls add-policy -domain hello.com")
	}

	fmt.Printf("➕ Adding TLS policy for: %s\n", *domain)
	fmt.Printf("   Challenge: %s\n", *challenge)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	var policy caddy.AutomationPolicy
	if *challenge == "http" {
		policy = caddy.NewSingleDomainHTTP01Policy(*domain)
	} else {
		return fmt.Errorf("DNS challenge requires dnspod-token (腾讯云 SecretId,SecretKey) configuration. Use 'domain add' instead")
	}

	if err := client.AddTLSPolicy(policy); err != nil {
		return fmt.Errorf("failed to add TLS policy: %w", err)
	}

	fmt.Println("✅ TLS policy added successfully")
	fmt.Printf("   Policy ID: %s\n", policy.ID)

	return nil
}

// runTLSRemovePolicy 移除 TLS 策略
func (c *caddyCmd) runTLSRemovePolicy(args []string) error {
	removeCmd := flag.NewFlagSet("remove-policy", flag.ExitOnError)
	adminAPI := removeCmd.String("admin", "http://127.0.0.1:2019", "Caddy Admin API address")
	policyID := removeCmd.String("id", "", "Policy ID (required)")

	if err := removeCmd.Parse(args); err != nil {
		return err
	}

	if *policyID == "" {
		return fmt.Errorf("policy ID is required. Use: hugov caddy tls remove-policy -id custom-hello-com")
	}

	fmt.Printf("🗑️  Removing TLS policy: %s\n", *policyID)

	config := &caddy.Config{
		AdminAPI: *adminAPI,
	}
	client := caddy.NewClient(config)

	if err := client.RemoveTLSPolicy(*policyID); err != nil {
		return fmt.Errorf("failed to remove TLS policy: %w", err)
	}

	fmt.Println("✅ TLS policy removed successfully")

	return nil
}
