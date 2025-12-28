package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
)

func main() {
	fmt.Println("🚀 Caddy Infrastructure Client Demo")
	fmt.Println("====================================\n")

	// 1. 创建 Caddy 客户端配置
	config := &caddy.Config{
		AdminAPI:       "http://127.0.0.1:2019",
		BinaryPath:     "caddy",
		DefaultBackend: "127.0.0.1:1314",
		CoreDomain:     "mdfriday.site",
	}

	client := caddy.NewClient(config)

	// 2. 启动 Caddy 服务器
	fmt.Println("📦 Starting Caddy server...")
	if err := client.StartServer(); err != nil {
		log.Fatalf("❌ Failed to start Caddy: %v", err)
	}
	defer func() {
		fmt.Println("\n🛑 Stopping Caddy server...")
		if err := client.Stop(); err != nil {
			log.Printf("Warning: Failed to stop Caddy cleanly: %v", err)
		}
	}()

	fmt.Println("✅ Caddy server started successfully")
	fmt.Printf("   - Admin API: %s\n", config.AdminAPI)
	fmt.Printf("   - Core domain: %s -> %s\n", config.CoreDomain, config.DefaultBackend)

	// 等待服务器完全启动
	time.Sleep(2 * time.Second)

	// 3. 验证服务器运行状态
	fmt.Println("\n🔍 Checking Caddy status...")
	if err := client.Ping(); err != nil {
		log.Fatalf("❌ Caddy is not responding: %v", err)
	}
	fmt.Println("✅ Caddy is running and responding")

	// 4. 获取当前配置
	fmt.Println("\n📋 Getting current configuration...")
	currentConfig, err := client.GetConfig()
	if err != nil {
		log.Printf("⚠️  Failed to get config: %v", err)
	} else {
		fmt.Printf("✅ Configuration retrieved (%d top-level keys)\n", len(currentConfig))
	}

	// 5. 添加静态站点示例
	fmt.Println("\n📁 Adding static sites...")

	sites := []struct {
		domain string
		path   string
	}{
		{"example.com", "/web/sites/example-com"},
		{"blog.example.com", "/web/sites/blog"},
		{"docs.example.com", "/web/sites/docs"},
	}

	for _, site := range sites {
		fmt.Printf("   Adding: %s -> %s\n", site.domain, site.path)
		if err := client.AddStaticSite(site.domain, site.path); err != nil {
			log.Printf("   ⚠️  Failed to add %s: %v", site.domain, err)
		} else {
			fmt.Printf("   ✅ Added: %s\n", site.domain)
		}
	}

	// 6. 等待一段时间，让 Caddy 尝试申请证书
	fmt.Println("\n⏳ Waiting for SSL certificates (5 seconds)...")
	time.Sleep(5 * time.Second)

	// 7. 查询证书状态
	fmt.Println("\n📜 Checking SSL certificate status...")
	for _, site := range sites {
		certInfo, err := client.GetCertificateStatus(site.domain)
		if err != nil {
			log.Printf("   ⚠️  Failed to check %s: %v", site.domain, err)
			continue
		}

		statusEmoji := "❓"
		switch certInfo.Status {
		case "issued":
			statusEmoji = "✅"
		case "pending":
			statusEmoji = "⏳"
		case "not_found":
			statusEmoji = "🔍"
		case "failed":
			statusEmoji = "❌"
		}

		fmt.Printf("   %s %s: %s\n", statusEmoji, site.domain, certInfo.Status)

		if certInfo.Status == "issued" {
			fmt.Printf("      Issuer: %s\n", certInfo.Issuer)
			fmt.Printf("      Valid: %s - %s\n",
				certInfo.NotBefore.Format("2006-01-02"),
				certInfo.NotAfter.Format("2006-01-02"))
		}
	}

	// 8. 移除一个站点示例
	fmt.Println("\n🗑️  Removing a site...")
	if err := client.RemoveStaticSite("blog.example.com"); err != nil {
		log.Printf("   ⚠️  Failed to remove site: %v", err)
	} else {
		fmt.Println("   ✅ Removed: blog.example.com")
	}

	// 9. 显示最终配置
	fmt.Println("\n📊 Final configuration summary...")
	finalConfig, err := client.GetConfig()
	if err != nil {
		log.Printf("⚠️  Failed to get final config: %v", err)
	} else {
		if apps, ok := finalConfig["apps"].(map[string]interface{}); ok {
			if httpApp, ok := apps["http"].(map[string]interface{}); ok {
				if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
					if mainServer, ok := servers["main"].(map[string]interface{}); ok {
						if routes, ok := mainServer["routes"].([]interface{}); ok {
							fmt.Printf("   Total routes: %d\n", len(routes))
						}
					}
				}
			}
		}
	}

	// 10. 保持运行，等待中断信号
	fmt.Println("\n✨ Demo complete! Caddy is still running.")
	fmt.Println("   Try accessing: http://mdfriday.site (if DNS is configured)")
	fmt.Println("   Admin API: http://127.0.0.1:2019/config/")
	fmt.Println("\nPress Ctrl+C to stop...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\n👋 Shutting down...")
}

