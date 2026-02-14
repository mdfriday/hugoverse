package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/mdfriday/hugoverse/internal/infrastructure/licensekit"
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
	if !licensekit.IsValidPlan(*plan) {
		validPlans := strings.Join(licensekit.GetValidPlans(), "|")
		return fmt.Errorf("invalid plan: %s (must be: %s)", *plan, validPlans)
	}

	// 获取 plan 配置
	planConfig := licensekit.GetPlanConfig(*plan)

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
		licenseKey := licensekit.GenerateLicenseKey()

		fmt.Printf("   [%d/%d] Creating: %s\n", i+1, *count, licenseKey)

		// 生成邮箱和密码
		email := licensekit.LicenseKeyToEmail(licenseKey)
		password := licensekit.LicenseKeyToPassword(licenseKey)

		// 步骤 1: 创建用户
		fmt.Printf("        → Creating user: %s\n", email)
		userErr := licensekit.CreateUser(*apiBase, email, password)
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
func (cmd *licenseCmd) createLicense(apiBase, token, licenseKey, plan string, planConf licensekit.PlanConfig) error {
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
