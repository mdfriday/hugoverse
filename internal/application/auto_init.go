package application

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdfriday/hugoverse/internal/domain/admin/entity"
	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
	"github.com/mdfriday/hugoverse/internal/infrastructure/licensekit"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// LicenseChecker 接口定义
type LicenseChecker interface {
	GetLicenseCount() int
	HasAnyLicense() bool
}

// AutoInitialize 从环境变量自动初始化系统
func AutoInitialize(adminApp *entity.Admin, db SystemDatabase, contentApp LicenseChecker, log loggers.Logger) error {
	// 1. 检查是否已初始化
	if db.SystemInitComplete() {
		log.Println("✅ System already initialized")

		// 检查企业功能是否已配置（通过 License 数量判断）
		licenseCount := contentApp.GetLicenseCount()
		hasLicense := licenseCount > 0

		if hasLicense {
			log.Printf("ℹ️  Found %d license(s) in database", licenseCount)
		}

		// 检查 Caddy 配置是否完整
		caddyConfigured, err := isCaddyProperlyConfigured(log)
		if err != nil {
			log.Warnf("⚠️  Failed to check Caddy configuration: %v", err)
			caddyConfigured = false
		}

		// 检查是否需要强制重新配置企业功能
		forceReconfigure := os.Getenv("FORCE_RECONFIGURE_ENTERPRISE") == "true"

		if forceReconfigure {
			log.Println("⚠️  FORCE_RECONFIGURE_ENTERPRISE=true detected")
			log.Println("   Re-configuring enterprise features...")
			go delayedEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log)
		} else if !hasLicense {
			// 系统已初始化，但没有 License，可能是企业功能配置失败
			log.Println("⚠️  No licenses found, enterprise features may not be configured")
			log.Println("   Attempting to configure enterprise features...")
			go delayedEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log)
		} else if !caddyConfigured {
			// 有 License 但 Caddy 配置不完整
			// 这种情况可能发生在：data/caddy 目录被清理但数据库保留的情况下
			// Caddy 从 Caddyfile.initial 启动（只有 503），但 Hugoverse 数据库中有完整的用户/License
			log.Println("⚠️  Caddy configuration incomplete (only default routes found)")
			log.Println("   This can happen if data/caddy was deleted but database was preserved")
			log.Println("   Re-initializing Caddy routes and enterprise site...")
			go delayedCaddyAndSiteReinitialize(adminApp, log)
		} else {
			// 系统已初始化、有 License、Caddy 配置完整，跳过企业功能配置
			// Caddy 已通过 --resume 从 autosave.json 恢复完整配置（包括动态添加的路由）
			log.Println("   ℹ️  Enterprise features already configured")
			log.Println("   ℹ️  Caddy configuration is complete")
			log.Println("   💡 To force reconfigure: set FORCE_RECONFIGURE_ENTERPRISE=true")
		}

		return nil
	}

	// 2. 检查是否启用自动初始化
	if os.Getenv("AUTO_INIT") != "true" {
		log.Println("ℹ️  AUTO_INIT is not enabled")
		log.Println("   Please visit /admin/init to configure manually")
		log.Printf("📍 Configuration URL: http://%s/admin/init", getEnvOrDefault("DOMAIN", "localhost"))
		return nil
	}

	log.Println("🔧 AUTO_INIT=true, initializing from environment variables...")

	// 3. 验证必需的环境变量
	if err := validateRequiredEnv(); err != nil {
		return err
	}

	// 4. 构建配置表单数据
	formData := buildConfigFormData(adminApp)

	// 5. 创建管理员用户
	email := strings.ToLower(os.Getenv("ADMIN_EMAIL"))
	log.Printf("👤 Creating admin user: %s", email)
	_, err := adminApp.NewUser(email, os.Getenv("ADMIN_PASSWORD"))
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// 6. 保存配置
	log.Println("💾 Saving system configuration...")
	err = adminApp.SetConfig(formData)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Println("✅ System initialization completed!")
	log.Printf("🌐 Admin panel: http://%s/admin", os.Getenv("DOMAIN"))
	log.Printf("🔒 Admin login: %s", email)

	// 注意：adminApp.Conf.AdminEmail 已被更新，handler 中的 RefreshAdmin 会动态读取最新值

	// 7. 等待配置生效
	time.Sleep(2 * time.Second)

	// 8. 配置企业功能（延迟执行，等待服务器启动）
	// 只在首次初始化时执行
	go delayedEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log)

	return nil
}

// delayedEnterpriseFeaturesWithLicenseCheck 延迟执行企业功能配置（带 License 检查）
// 在独立的 goroutine 中运行，等待服务器启动后再执行
func delayedEnterpriseFeaturesWithLicenseCheck(adminApp *entity.Admin, contentApp LicenseChecker, log loggers.Logger) {
	// 等待一段时间，确保服务器已经启动
	log.Println("⏳ Waiting for server to start before configuring enterprise features...")
	time.Sleep(5 * time.Second)

	if err := configureEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log); err != nil {
		log.Warnf("⚠️  Failed to configure enterprise features: %v", err)
		log.Println("   You can configure them manually later")
	}
}

// delayedEnterpriseFeatures 延迟执行企业功能配置（向后兼容，已废弃）
// 在独立的 goroutine 中运行，等待服务器启动后再执行
// 已废弃：请使用 delayedEnterpriseFeaturesWithLicenseCheck
func delayedEnterpriseFeatures(adminApp *entity.Admin, log loggers.Logger) {
	// 等待一段时间，确保服务器已经启动
	log.Println("⏳ Waiting for server to start before configuring enterprise features...")
	time.Sleep(5 * time.Second)

	if err := configureEnterpriseFeatures(adminApp, log); err != nil {
		log.Warnf("⚠️  Failed to configure enterprise features: %v", err)
		log.Println("   You can configure them manually later")
	}
}

// validateRequiredEnv 验证必需的环境变量
func validateRequiredEnv() error {
	required := map[string]string{
		"DOMAIN":           os.Getenv("DOMAIN"),
		"ADMIN_EMAIL":      os.Getenv("ADMIN_EMAIL"),
		"ADMIN_PASSWORD":   os.Getenv("ADMIN_PASSWORD"),
		"COUCHDB_PASSWORD": os.Getenv("COUCHDB_PASSWORD"),
	}

	var missing []string
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// buildConfigFormData 构建配置表单数据
func buildConfigFormData(adminApp *entity.Admin) url.Values {
	formData := url.Values{}

	// 基础配置
	formData.Set("name", getEnvOrDefault("SITE_NAME", "Hugoverse"))
	formData.Set("domain", os.Getenv("DOMAIN"))
	formData.Set("server_ip", os.Getenv("SERVER_IP"))

	// 管理员配置
	email := strings.ToLower(os.Getenv("ADMIN_EMAIL"))
	formData.Set("email", email)
	formData.Set("password", os.Getenv("ADMIN_PASSWORD"))
	formData.Set("admin_email", email)

	// CouchDB 配置
	formData.Set("url", getEnvOrDefault("COUCHDB_URL", "http://couchdb:5984"))
	formData.Set("admin_user", getEnvOrDefault("COUCHDB_USER", "admin"))
	formData.Set("admin_pass", os.Getenv("COUCHDB_PASSWORD"))
	formData.Set("db_prefix", getEnvOrDefault("COUCHDB_DB_PREFIX", "userdb-"))

	// Caddy 配置
	formData.Set("caddy_host", getEnvOrDefault("CADDY_HOST", "caddy"))
	formData.Set("caddy_port", getEnvOrDefault("CADDY_PORT", "2019"))

	// 生成安全令牌
	name := []byte(formData.Get("name") + adminApp.NewETage())
	secret := base64.StdEncoding.EncodeToString(name)
	formData.Set("client_secret", secret)
	formData.Set("etag", adminApp.NewETage())

	// 设置 HTTP 端口
	formData.Set("http_port", adminApp.HttpPort())

	return formData
}

// configureEnterpriseFeaturesWithLicenseCheck 配置企业功能（带 License 检查）
func configureEnterpriseFeaturesWithLicenseCheck(adminApp *entity.Admin, contentApp LicenseChecker, log loggers.Logger) error {
	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🏢 Configuring Enterprise Features")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	// 1. 初始化 Caddy 路由
	if err := initializeCaddyRoutes(log); err != nil {
		log.Warnf("⚠️  Failed to initialize Caddy: %v", err)
		// 不返回错误，继续其他配置
	}

	// 2. 自动配置企业站点（先于 License：与「先 caddy start 再 caddy add」一致，且避免长时间等待 API 时 Caddy 无企业路由）
	if os.Getenv("AUTO_CONFIGURE_ENTERPRISE_SITE") == "true" {
		if err := configureEnterpriseSite(log); err != nil {
			log.Warnf("⚠️  Failed to configure enterprise site: %v", err)
			// 不返回错误，用户可以手动配置
		}
	}

	// 3. 检查 License 是否已存在
	licenseCount := contentApp.GetLicenseCount()
	if licenseCount > 0 {
		log.Printf("ℹ️  Found %d existing license(s), skipping license generation", licenseCount)
		log.Println("   💡 To force regenerate: set FORCE_RECONFIGURE_ENTERPRISE=true")
	} else {
		// 4. 自动生成企业 License（如果启用且不存在）
		if os.Getenv("AUTO_GENERATE_ENTERPRISE_LICENSE") == "true" {
			if err := generateEnterpriseLicense(log); err != nil {
				log.Warnf("⚠️  Failed to generate enterprise license: %v", err)
				// 不返回错误，用户可以手动生成
			}
		}
	}

	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("✅ Enterprise Features Configuration Complete")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	return nil
}

// configureEnterpriseFeatures 配置企业功能（向后兼容，已废弃）
// 已废弃：请使用 configureEnterpriseFeaturesWithLicenseCheck
func configureEnterpriseFeatures(adminApp *entity.Admin, log loggers.Logger) error {
	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🏢 Configuring Enterprise Features")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	// 1. 初始化 Caddy 路由
	if err := initializeCaddyRoutes(log); err != nil {
		log.Warnf("⚠️  Failed to initialize Caddy: %v", err)
		// 不返回错误，继续其他配置
	}

	// 2. 自动配置企业站点（先于 License：与「先 caddy start 再 caddy add」一致，且避免长时间等待 API 时 Caddy 无企业路由）
	if os.Getenv("AUTO_CONFIGURE_ENTERPRISE_SITE") == "true" {
		if err := configureEnterpriseSite(log); err != nil {
			log.Warnf("⚠️  Failed to configure enterprise site: %v", err)
			// 不返回错误，用户可以手动配置
		}
	}

	// 3. 自动生成企业 License（如果启用）
	if os.Getenv("AUTO_GENERATE_ENTERPRISE_LICENSE") == "true" {
		if err := generateEnterpriseLicense(log); err != nil {
			log.Warnf("⚠️  Failed to generate enterprise license: %v", err)
			// 不返回错误，用户可以手动生成
		}
	}

	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("✅ Enterprise Features Configuration Complete")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	return nil
}

// initializeCaddyRoutes 初始化 Caddy 基础路由
func initializeCaddyRoutes(log loggers.Logger) error {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		return fmt.Errorf("DOMAIN is not set")
	}

	log.Println("🔧 Initializing Caddy routes...")

	// 获取配置
	caddyAdminAPI := getEnvOrDefault("CADDY_ADMIN_API", "http://caddy:2019")
	dnspodToken := ""
	isLocalhost := (domain == "localhost" || domain == "127.0.0.1")

	// 对于非 localhost，配置 DNSPod
	if !isLocalhost && os.Getenv("DNSPOD_ENABLED") == "true" {
		dnspodID := os.Getenv("DNSPOD_ID")
		dnspodSecret := os.Getenv("DNSPOD_SECRET")
		if dnspodID != "" && dnspodSecret != "" {
			dnspodToken = fmt.Sprintf("%s,%s", dnspodID, dnspodSecret)
		}
	}

	// 创建 Caddy 客户端
	config := &caddy.Config{
		AdminAPI:       caddyAdminAPI,
		DefaultBackend: "hugoverse:1314",
		CouchDBBackend: "couchdb:5984",
		CoreDomain:     domain,
		DNSPodToken:    dnspodToken,
		ServerIP:       os.Getenv("SERVER_IP"),
		ConfigPath:     "/tmp/caddy-config.json",
	}
	// localhost 且自动挂企业站：核心走 app.localhost，apex 给静态站（与生产 app. / 根域 一致）
	if isLocalhost && os.Getenv("AUTO_CONFIGURE_ENTERPRISE_SITE") == "true" {
		config.LocalDevUseAppSubdomain = true
	}

	client := caddy.NewClient(config)

	// 等待 Caddy 就绪
	log.Println("   Waiting for Caddy Admin API...")
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if err := client.Ping(); err == nil {
			break
		}
		if i == maxRetries-1 {
			return fmt.Errorf("Caddy Admin API not responding after %d retries", maxRetries)
		}
		time.Sleep(time.Second)
	}

	log.Println("✅ Caddy is ready")

	// 配置基础路由（通过 Admin API），与 hugov caddy start 行为对齐
	log.Println("   Configuring routes via Admin API...")

	if isLocalhost {
		log.Println("   Configuring localhost routes (HTTP :80)...")
		if err := client.ConfigureLocalhost(); err != nil {
			log.Warnf("⚠️  Failed to configure localhost routes: %v", err)
		} else {
			log.Println("   ✅ Localhost routes configured")
		}
		if config.LocalDevUseAppSubdomain {
			log.Printf("   Core / Admin (dev): http://app.%s  — 宿主机可执行: echo '127.0.0.1 app.%s' | sudo tee -a /etc/hosts", domain, domain)
			log.Printf("   Enterprise static (apex): http://%s", domain)
		} else {
			log.Printf("   Core (dev): http://%s", domain)
		}
		log.Printf("   CouchDB (dev): http://cdb.%s", domain)
	} else {
		log.Println("   Configuring production routes (app. / cdb. / optional wildcard)...")
		if err := client.ConfigureProductionPlatform(); err != nil {
			log.Warnf("⚠️  Failed to configure production Caddy routes: %v", err)
		} else {
			log.Println("   ✅ Production routes configured (same as hugov caddy start)")
		}
		log.Printf("   Hugoverse API: https://app.%s (HTTP: http://app.%s)", domain, domain)
		log.Printf("   CouchDB proxy: https://cdb.%s (HTTP: http://cdb.%s)", domain, domain)
		log.Printf("   Enterprise static: use root domain (hugov caddy add -domain %s -path ...)", domain)
		if dnspodToken != "" {
			log.Printf("   Wildcard TLS: *.%s (DNSPod / tencentcloud)", domain)
		}
	}

	return nil
}

// generateEnterpriseLicense 自动生成企业 License
func generateEnterpriseLicense(log loggers.Logger) error {
	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("📋 Enterprise License Generation")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	// 1. 获取 Master License
	masterLicenseKey := os.Getenv("MASTER_LICENSE")

	if masterLicenseKey == "" {
		log.Println("ℹ️  No Master License provided")
		log.Println("   Using FREE mode (1 enterprise license)")
	} else {
		log.Println("🔑 Master License provided")
		log.Println("   Verifying online...")
	}

	// 2. 在线验证 Master License
	masterInfo, err := licensekit.VerifyMasterLicenseOnline(masterLicenseKey)
	if err != nil {
		log.Warnf("⚠️  Master License verification failed: %v", err)
		log.Println("   Falling back to FREE mode (1 enterprise license)")
	}

	log.Println("")
	log.Println("📊 License Quota Information")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   Type: %s", masterInfo.Type)
	log.Printf("   Max Sub-Licenses: %d", masterInfo.MaxSubLicenses)
	log.Printf("   Used: %d", masterInfo.UsedLicenses)
	log.Printf("   Remaining: %d", masterInfo.GetRemainingQuota())
	if !masterInfo.ExpiryDate.IsZero() {
		log.Printf("   Expires: %s", masterInfo.ExpiryDate.Format("2006-01-02"))
	}

	if masterInfo.Type == "free" {
		log.Println("")
		log.Println("💡 Need more licenses?")
		log.Println("   Visit: https://mdfriday.com/pricing")
		log.Println("   Purchase a Master License and add to .env:")
		log.Println("   MASTER_LICENSE=YOUR_LICENSE_KEY")
	}
	log.Println("")

	// 3. 检查是否还能生成
	requestCount := 1
	if countStr := os.Getenv("ENTERPRISE_LICENSE_COUNT"); countStr != "" {
		fmt.Sscanf(countStr, "%d", &requestCount)
	}

	if requestCount < 1 {
		requestCount = 1
	}

	if !masterInfo.CanGenerateMore(requestCount) {
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Println("❌ License Quota Exceeded")
		log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("   Current: %d / %d licenses\n", masterInfo.UsedLicenses, masterInfo.MaxSubLicenses)
		log.Printf("   Requested: %d\n", requestCount)
		log.Printf("   Available: %d\n", masterInfo.GetRemainingQuota())
		log.Println("")
		log.Println("💡 To generate more licenses:")
		log.Println("   1. Visit: https://mdfriday.com/pricing")
		log.Println("   2. Purchase a Master License")
		log.Println("   3. Add to .env: MASTER_LICENSE=YOUR_KEY")
		log.Println("   4. Restart: docker-compose restart hugoverse")
		log.Println("")
		return fmt.Errorf("license quota exceeded")
	}

	// 4. 等待 API 就绪
	apiBase := "http://localhost:1314"
	log.Println("🔄 Waiting for Hugoverse API...")

	maxRetries := 60 // 增加到 60 次，每次 2 秒，总共 2 分钟
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(apiBase + "/api/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			log.Println("✅ API is ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if i == maxRetries-1 {
			return fmt.Errorf("Hugoverse API not responding after %d retries", maxRetries)
		}
		time.Sleep(2 * time.Second) // 增加到 2 秒
	}

	// 5. 登录获取 token
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	plan := getEnvOrDefault("ENTERPRISE_LICENSE_PLAN", "enterprise")

	log.Println("")
	log.Println("🔐 Logging in as admin...")
	token, err := licensekit.LoginAndGetToken(apiBase, adminEmail, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	log.Println("✅ Login successful")

	// 6. 生成 License
	log.Println("")
	log.Printf("🔨 Generating %d enterprise license(s)...\n", requestCount)
	log.Println("")

	planConfig := licensekit.GetPlanConfig(plan)
	successCount := 0

	type LicenseResult struct {
		Key      string
		Email    string
		Password string
	}

	results := []LicenseResult{}

	for i := 0; i < requestCount; i++ {
		// 使用预定义的 Key 或生成新 Key
		var licenseKey string
		if i == 0 && os.Getenv("ENTERPRISE_LICENSE_KEY") != "" {
			licenseKey = os.Getenv("ENTERPRISE_LICENSE_KEY")
		} else {
			licenseKey = licensekit.GenerateLicenseKey()
		}

		log.Printf("   [%d/%d] Creating: %s", i+1, requestCount, licenseKey)

		// 创建用户
		email := licensekit.LicenseKeyToEmail(licenseKey)
		password := licensekit.LicenseKeyToPassword(licenseKey)

		if err := licensekit.CreateUser(apiBase, email, password); err != nil {
			log.Printf("      ⚠️  User creation failed: %v", err)
		} else {
			log.Printf("      ✅ User created: %s", email)
		}

		// 创建 License
		if err := licensekit.CreateLicenseViaAPI(apiBase, token, licenseKey, plan, planConfig); err != nil {
			log.Printf("      ❌ License creation failed: %v", err)
		} else {
			log.Printf("      ✅ License created")
			successCount++
			results = append(results, LicenseResult{
				Key:      licenseKey,
				Email:    email,
				Password: password,
			})
		}
		log.Println("")
	}

	// 7. 上报使用情况（仅付费版）
	if masterLicenseKey != "" && masterLicenseKey != "FREE" && successCount > 0 {
		log.Println("📊 Reporting usage to license server...")
		if err := licensekit.ReportUsage(masterLicenseKey, successCount); err != nil {
			log.Warnf("⚠️  Failed to report usage: %v", err)
			log.Println("   (This won't affect your licenses)")
		} else {
			log.Println("✅ Usage reported")
		}
		log.Println("")
	}

	// 8. 显示结果
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("📋 Generation Summary")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   Requested: %d\n", requestCount)
	log.Printf("   Success: %d\n", successCount)
	log.Printf("   Failed: %d\n", requestCount-successCount)
	log.Println("")

	if len(results) > 0 {
		log.Println("✅ Generated Licenses:")
		log.Println("")
		for i, result := range results {
			log.Printf("   %d. License Key: %s\n", i+1, result.Key)
			log.Printf("      Email: %s\n", result.Email)
			log.Printf("      Password: %s\n", result.Password)
			log.Println("")
		}

		log.Println("💡 Important:")
		log.Println("   - Save these credentials securely")
		log.Println("   - Share with your users")
		log.Println("   - Users activate in Obsidian Friday plugin")
		log.Println("")
	}

	return nil
}

// configureEnterpriseSite 自动配置企业站点
func configureEnterpriseSite(log loggers.Logger) error {
	log.Println("")
	log.Println("🌐 Configuring Enterprise Site...")

	domain := os.Getenv("ENTERPRISE_SITE_DOMAIN")
	if domain == "" {
		domain = os.Getenv("DOMAIN")
	}

	coreDomain := os.Getenv("DOMAIN")
	isLocalhost := coreDomain == "localhost" || coreDomain == "127.0.0.1"

	sitePath := os.Getenv("ENTERPRISE_SITE_PATH")
	if sitePath == "" {
		sitePath = filepath.Join(getEnvOrDefault("HUGOVERSE_DATA_DIR", "/data"), "enterprise")
	}

	// 确保目录存在
	if err := EnsureDirExists(sitePath); err != nil {
		return fmt.Errorf("failed to create site directory: %w", err)
	}

	log.Printf("   Site host (apex): %s — same as hugov caddy add -domain %s -path <dir>", domain, domain)
	log.Printf("   Path (Hugoverse): %s", sitePath)
	if mapped := caddy.ToCaddySiteRootPath(sitePath); mapped != sitePath {
		log.Printf("   Path (Caddy file_server root): %s (HUGOVERSE_DATA_DIR → CADDY_SITE_ROOT)", mapped)
	}

	// 与 initializeCaddyRoutes 一致，便于 AddStaticSite 处理通配符 route（wildcard-%s 依赖 CoreDomain）
	caddyAdminAPI := getEnvOrDefault("CADDY_ADMIN_API", "http://caddy:2019")
	dnspodToken := ""
	if !isLocalhost && os.Getenv("DNSPOD_ENABLED") == "true" {
		id, sec := os.Getenv("DNSPOD_ID"), os.Getenv("DNSPOD_SECRET")
		if id != "" && sec != "" {
			dnspodToken = fmt.Sprintf("%s,%s", id, sec)
		}
	}
	config := &caddy.Config{
		AdminAPI:       caddyAdminAPI,
		CoreDomain:     coreDomain,
		DNSPodToken:    dnspodToken,
		DefaultBackend: "hugoverse:1314",
		CouchDBBackend: "couchdb:5984",
	}

	client := caddy.NewClient(config)

	// 等价于 hugov caddy add -domain <apex> -path <path>（企业站挂在根域名，与 app./cdb. 子域分离）
	log.Println("   Adding static site to Caddy...")
	var addErr error
	for attempt := 1; attempt <= 30; attempt++ {
		addErr = client.AddStaticSite(domain, sitePath)
		if addErr == nil {
			break
		}
		// ConfigureLocalhost / ConfigureProductionPlatform 后 Caddy 可能短暂重载，Admin API 会 connection refused
		if attempt < 30 && strings.Contains(addErr.Error(), "connection refused") {
			log.Printf("   Caddy Admin API reloading, retry %d/30...", attempt)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if addErr != nil {
		return fmt.Errorf("failed to add site: %w", addErr)
	}

	log.Println("✅ Enterprise site configured")
	if isLocalhost {
		log.Printf("   Access at: http://%s", domain)
	} else {
		log.Printf("   Access at: https://%s (and http://%s)", domain, domain)
	}

	return nil
}

// SystemDatabase 接口定义
type SystemDatabase interface {
	SystemInitComplete() bool
}

// LicenseChecker 接口定义（已移到文件顶部，此处保留注释以便理解）
// type LicenseChecker interface {
//     GetLicenseCount() int
//     HasAnyLicense() bool
// }

// isCaddyProperlyConfigured 检查 Caddy 配置是否完整
// 通过查询 Caddy Admin API 检查是否包含必要的路由（不仅仅是默认的 503 响应）
func isCaddyProperlyConfigured(log loggers.Logger) (bool, error) {
	caddyAdminAPI := os.Getenv("CADDY_ADMIN_API")
	if caddyAdminAPI == "" {
		caddyAdminAPI = "http://caddy:2019"
	}

	// 查询当前 Caddy 配置
	resp, err := http.Get(caddyAdminAPI + "/config/")
	if err != nil {
		return false, fmt.Errorf("failed to query Caddy config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Caddy returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read Caddy config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return false, fmt.Errorf("failed to parse Caddy config: %w", err)
	}

	// 检查是否有 HTTP 服务器配置
	apps, ok := config["apps"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	httpApp, ok := apps["http"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	servers, ok := httpApp["servers"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	// 检查所有服务器的路由
	for _, server := range servers {
		serverMap, ok := server.(map[string]interface{})
		if !ok {
			continue
		}

		routes, ok := serverMap["routes"].([]interface{})
		if !ok || len(routes) == 0 {
			continue
		}

		// 如果只有一个路由，检查是否是默认的 503 响应
		if len(routes) == 1 {
			route, ok := routes[0].(map[string]interface{})
			if !ok {
				continue
			}

			handlers, ok := route["handle"].([]interface{})
			if !ok || len(handlers) != 1 {
				continue
			}

			handler, ok := handlers[0].(map[string]interface{})
			if !ok {
				continue
			}

			// 检查是否是 503 初始化响应
			if handler["handler"] == "static_response" {
				statusCode, ok := handler["status_code"].(float64)
				if ok && statusCode == 503 {
					// 只有默认的 503 响应，配置不完整
					log.Println("   🔍 Caddy only has default 503 route")
					return false, nil
				}
			}
		}

		// 有多个路由或非 503 路由，认为配置完整
		log.Printf("   ✅ Caddy has %d route(s) configured", len(routes))
		return true, nil
	}

	// 没有找到合适的路由配置
	return false, nil
}

// delayedCaddyAndSiteReinitialize 延迟重新初始化 Caddy 路由和企业站点
// 用于修复 Caddy 配置丢失的情况（例如 data 目录被清理后）
func delayedCaddyAndSiteReinitialize(adminApp *entity.Admin, log loggers.Logger) {
	// 等待服务器启动
	time.Sleep(3 * time.Second)

	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🔧 Re-initializing Caddy & Enterprise Site")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")

	// 1. 重新配置 Caddy 路由
	if err := initializeCaddyRoutes(log); err != nil {
		log.Warnf("⚠️  Failed to configure Caddy routes: %v", err)
	}

	// 2. 重新配置企业站点
	if err := configureEnterpriseSite(log); err != nil {
		log.Warnf("⚠️  Failed to configure enterprise site: %v", err)
	}

	log.Println("")
	log.Println("✅ Caddy and Enterprise Site re-initialized")
	log.Println("")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
