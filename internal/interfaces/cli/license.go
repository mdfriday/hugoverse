package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type licenseCmd struct {
	parent *flag.FlagSet
	cmd    *flag.FlagSet
}

func NewLicenseCmd(parent *flag.FlagSet) (*licenseCmd, error) {
	nCmd := &licenseCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("license", flag.ExitOnError)
	nCmd.cmd.Usage = func() {
		fmt.Println("Usage: hugov license generate [options]")
		fmt.Println("\nDescription:")
		fmt.Println("  Batch generate license keys and create them in database")
		fmt.Println("\nOptions:")
		fmt.Println("  -email        Email for login (required)")
		fmt.Println("  -password     Password for login (required)")
		fmt.Println("  -api          API base URL (default: http://127.0.0.1:1314)")
		fmt.Println("  -plan         License plan: free|starter|creator|pro|enterprise (required)")
		fmt.Println("  -count        Number of licenses to generate (default: 1)")
		fmt.Println("\nPlan Details:")
		fmt.Println("  free       - 30 days, 1 device, 1 IP")
		fmt.Println("  starter    - 365 days, 3 devices, 3 IPs")
		fmt.Println("  creator    - 365 days, 5 devices, 5 IPs")
		fmt.Println("  pro        - 365 days, 10 devices, 10 IPs")
		fmt.Println("  enterprise - 36500 days, 999 devices, 999 IPs")
		fmt.Println("\nExample:")
		fmt.Println("  hugov license generate \\")
		fmt.Println("    -email mdf_public@mdfriday.com \\")
		fmt.Println("    -password 987123 \\")
		fmt.Println("    -plan starter \\")
		fmt.Println("    -count 5")
		fmt.Println("\n  This will generate 5 starter licenses with random keys")
	}

	err := nCmd.cmd.Parse(parent.Args()[1:])
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (cmd *licenseCmd) Usage() {
	cmd.cmd.Usage()
}

func (cmd *licenseCmd) Run() error {
	if len(cmd.cmd.Args()) == 0 {
		cmd.Usage()
		return errors.New("please specify 'generate' subcommand")
	}

	subCommand := cmd.cmd.Args()[0]

	switch subCommand {
	case "generate":
		return cmd.runGenerate(cmd.cmd.Args()[1:])
	default:
		cmd.Usage()
		return fmt.Errorf("invalid license subcommand: %s (only 'generate' is supported)", subCommand)
	}
}

// runGenerate 批量生成 license
func (cmd *licenseCmd) runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	email := fs.String("email", "", "Email for login")
	password := fs.String("password", "", "Password for login")
	apiBase := fs.String("api", "http://127.0.0.1:1314", "API base URL")
	plan := fs.String("plan", "", "License plan (free|starter|creator|pro|lifetime)")
	count := fs.Int("count", 1, "Number of licenses to generate")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 验证必填参数
	if *email == "" {
		return errors.New("email is required (-email)")
	}
	if *password == "" {
		return errors.New("password is required (-password)")
	}
	if *plan == "" {
		return errors.New("plan is required (-plan)")
	}
	if *count < 1 {
		return errors.New("count must be at least 1")
	}

	// 验证 plan 类型
	validPlans := map[string]bool{
		"free": true, "starter": true, "creator": true, "pro": true, "enterprise": true,
	}
	if !validPlans[*plan] {
		return fmt.Errorf("invalid plan: %s (must be: free|starter|creator|pro|enterprise)", *plan)
	}

	// 获取 plan 配置
	planConfig := cmd.getPlanConfig(*plan)

	fmt.Println("🚀 Batch License Generation")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Email: %s\n", *email)
	fmt.Printf("   API: %s\n", *apiBase)
	fmt.Printf("   Plan: %s\n", *plan)
	fmt.Printf("   Count: %d\n", *count)
	fmt.Printf("   Max Devices: %d\n", planConfig.MaxDevices)
	fmt.Printf("   Max IPs: %d\n", planConfig.MaxIPs)
	fmt.Println()

	// 第一步：登录获取 token
	fmt.Println("📝 Step 1: Logging in...")
	token, err := cmd.login(*apiBase, *email, *password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Printf("✅ Login successful\n\n")

	// 第二步：批量生成并创建 license
	fmt.Printf("📝 Step 2: Generating %d licenses...\n", *count)
	fmt.Println()

	successCount := 0
	failCount := 0
	generatedKeys := []string{}

	// 记录用户创建信息
	type LicenseInfo struct {
		Key      string
		Email    string
		Password string
	}
	licenseInfos := []LicenseInfo{}

	for i := 0; i < *count; i++ {
		// 生成 license key
		licenseKey := cmd.generateLicenseKey(*plan)

		fmt.Printf("   [%d/%d] Creating: %s\n", i+1, *count, licenseKey)

		// 生成邮箱和密码
		email := cmd.licenseKeyToEmail(licenseKey)
		password := cmd.licenseKeyToPassword(licenseKey)

		// 步骤 1: 创建用户
		fmt.Printf("        → Creating user: %s\n", email)
		userErr := cmd.createUser(*apiBase, email, password)
		if userErr != nil {
			fmt.Printf("        ❌ User creation failed: %v\n", userErr)
			failCount++
			continue // 用户创建失败，跳过 license 创建
		}
		fmt.Printf("        ✅ User created\n")

		// 步骤 2: 创建 license
		fmt.Printf("        → Creating license\n")
		licenseErr := cmd.createLicense(*apiBase, token, licenseKey, *plan, planConfig)

		if licenseErr != nil {
			fmt.Printf("        ❌ License creation failed: %v\n", licenseErr)
			failCount++
		} else {
			fmt.Printf("        ✅ License created\n")
			successCount++
			generatedKeys = append(generatedKeys, licenseKey)
			licenseInfos = append(licenseInfos, LicenseInfo{
				Key:      licenseKey,
				Email:    email,
				Password: password,
			})
		}
		fmt.Println() // 空行分隔每个 license
	}

	// 第三步：显示结果
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Generation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Total: %d\n", *count)
	fmt.Printf("   Success: %d\n", successCount)
	fmt.Printf("   Failed: %d\n", failCount)
	fmt.Println()

	if len(licenseInfos) > 0 {
		fmt.Println("✅ Generated Licenses:")
		fmt.Println()
		for i, info := range licenseInfos {
			fmt.Printf("   %d. License Key: %s\n", i+1, info.Key)
			fmt.Printf("      Email:       %s\n", info.Email)
			fmt.Printf("      Password:    %s\n", info.Password)
			fmt.Println()
		}
		fmt.Println("💡 Tips:")
		fmt.Println("   - Save these credentials in a secure location")
		fmt.Println("   - Users can login with Email + Password")
		fmt.Println("   - License Key is used for activation")
	}

	if failCount > 0 {
		return fmt.Errorf("%d license(s) failed to create", failCount)
	}

	fmt.Println("🎉 All licenses and users created successfully!")
	return nil
}

// PlanConfig 定义 plan 配置
type PlanConfig struct {
	// 设备/IP 限制
	MaxDevices int `json:"max_devices"`
	MaxIPs     int `json:"max_ips"`

	// Sync 功能
	SyncEnabled bool `json:"sync_enabled"`
	SyncQuotaMB int  `json:"sync_quota"`

	// Publish 功能
	PublishEnabled  bool `json:"publish_enabled"`
	MaxSites        int  `json:"max_sites"`
	MaxStorageMB    int  `json:"max_storage"`
	CustomDomain    bool `json:"custom_domain"`
	CustomSubDomain bool `json:"custom_sub_domain"` // 二级域名

	// 有效期（天数）
	ValidityDays int `json:"validity_days"`
}

// getPlanConfig 获取 plan 配置
func (cmd *licenseCmd) getPlanConfig(plan string) PlanConfig {
	configs := map[string]PlanConfig{
		"free": {
			MaxDevices:      3,
			MaxIPs:          3,
			SyncEnabled:     true,
			SyncQuotaMB:     500,
			PublishEnabled:  true,
			MaxSites:        3,
			MaxStorageMB:    10240, // 10G
			CustomSubDomain: true,
			CustomDomain:    true,
			ValidityDays:    3, // ✅ 3 天
		},
		"starter": {
			MaxDevices:      3,
			MaxIPs:          3,
			SyncEnabled:     true,
			SyncQuotaMB:     500,
			PublishEnabled:  false, // ❌ 不支持发布
			MaxSites:        0,
			MaxStorageMB:    1024, // 1G
			CustomSubDomain: false,
			CustomDomain:    false,
			ValidityDays:    365,
		},
		"enjoy": {
			MaxDevices:      5,
			MaxIPs:          5,
			SyncEnabled:     true,
			SyncQuotaMB:     2048,
			PublishEnabled:  false,
			MaxSites:        0,
			MaxStorageMB:    10240, // 10G
			CustomSubDomain: false,
			CustomDomain:    false,
			ValidityDays:    365,
		},
		"creator": {
			MaxDevices:      5,
			MaxIPs:          5,
			SyncEnabled:     true,
			SyncQuotaMB:     2048,
			PublishEnabled:  true,
			MaxSites:        10,
			MaxStorageMB:    10240, // 10G
			CustomSubDomain: true,  // ✅ 二级域名
			CustomDomain:    false,
			ValidityDays:    365,
		},
		"pro": {
			MaxDevices:      10,
			MaxIPs:          10,
			SyncEnabled:     true,
			SyncQuotaMB:     10240,
			PublishEnabled:  true,
			MaxSites:        50,
			MaxStorageMB:    10240, // 10G
			CustomSubDomain: true,
			CustomDomain:    true, // ✅ 独立域名
			ValidityDays:    365,
		},
		"enterprise": {
			MaxDevices:      100,
			MaxIPs:          100,
			SyncEnabled:     true,
			SyncQuotaMB:     51200,
			PublishEnabled:  true,
			MaxSites:        100,
			MaxStorageMB:    102400, // 100G
			CustomSubDomain: true,
			CustomDomain:    true,
			ValidityDays:    365 * 100, // ✅ 100 年
		},
	}

	return configs[plan]
}

// licenseKeyToEmail 将 License Key 转换为邮箱
// 规则：去掉 "MDF-" 前缀，转小写，加上 @mdfriday.com
func (cmd *licenseCmd) licenseKeyToEmail(licenseKey string) string {
	key := strings.ToLower(strings.TrimPrefix(licenseKey, "MDF-"))
	return fmt.Sprintf("%s@mdfriday.com", key)
}

// licenseKeyToPassword 将 License Key 转换为密码
// 规则：去掉 "MDF-" 前缀，转小写，base64 编码
func (cmd *licenseCmd) licenseKeyToPassword(licenseKey string) string {
	key := strings.ToLower(strings.TrimPrefix(licenseKey, "MDF-"))
	return base64.StdEncoding.EncodeToString([]byte(key))
}

// generateLicenseKey 生成 license key
// 格式：MDF-XXXX-XXXX-XXXX（全随机，避免被猜测）
func (cmd *licenseCmd) generateLicenseKey(plan string) string {
	// 生成三个随机部分，每部分 4 个字符
	part1 := cmd.generateRandomString(4)
	part2 := cmd.generateRandomString(4)
	part3 := cmd.generateRandomString(4)

	return fmt.Sprintf("MDF-%s-%s-%s", part1, part2, part3)
}

// generateRandomString 生成随机字符串
func (cmd *licenseCmd) generateRandomString(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符 0,O,1,I
	b := make([]byte, length)
	rand.Read(b)

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[int(b[i])%len(charset)]
	}

	return string(result)
}

// createUser 创建用户
func (cmd *licenseCmd) createUser(apiBase, email, password string) error {
	userURL := fmt.Sprintf("%s/api/user", apiBase)

	// 构造表单数据
	data := fmt.Sprintf("email=%s&password=%s", email, password)

	req, err := http.NewRequest("POST", userURL, strings.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 接受 200 OK 或 201 Created
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// login 登录获取 token
func (cmd *licenseCmd) login(apiBase, email, password string) (string, error) {
	loginURL := fmt.Sprintf("%s/api/login", apiBase)

	// 构造表单数据
	data := fmt.Sprintf("email=%s&password=%s", email, password)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 接受 200 OK 或 201 Created
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
		Error   string   `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	// 如果响应中有 data 字段且不为空，就认为登录成功
	if len(result.Data) > 0 {
		return result.Data[0], nil
	}

	// 检查 success 字段（兼容性）
	if result.Success && len(result.Data) == 0 {
		return "", fmt.Errorf("login successful but no token returned")
	}

	return "", fmt.Errorf("login failed: %s", result.Error)
}

// createLicense 创建 license
func (cmd *licenseCmd) createLicense(apiBase, token, licenseKey, plan string, planConf PlanConfig) error {
	createURL := fmt.Sprintf("%s/api/content?type=License", apiBase)

	// 构造 multipart 表单
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加字段
	fields := map[string]string{
		"id":          "-1",
		"license_key": licenseKey,
		"plan":        plan,
		"max_devices": fmt.Sprintf("%d", planConf.MaxDevices),
		"max_ips":     fmt.Sprintf("%d", planConf.MaxIPs),

		"sync_enabled": fmt.Sprintf("%t", planConf.SyncEnabled),
		"sync_quota":   fmt.Sprintf("%d", planConf.SyncQuotaMB),

		"publish_enabled":   fmt.Sprintf("%t", planConf.PublishEnabled),
		"max_sites":         fmt.Sprintf("%d", planConf.MaxSites),
		"max_storage":       fmt.Sprintf("%d", planConf.MaxStorageMB),
		"custom_domain":     fmt.Sprintf("%t", planConf.CustomDomain),
		"custom_sub_domain": fmt.Sprintf("%t", planConf.CustomSubDomain),

		"validity_days": fmt.Sprintf("%d", planConf.ValidityDays),
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	// 创建请求
	req, err := http.NewRequest("POST", createURL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
