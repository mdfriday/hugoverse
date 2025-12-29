package cli

import (
	"bytes"
	"crypto/rand"
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
	fmt.Printf("   Expiry: %d days\n", planConfig.ExpiryDays)
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

	for i := 0; i < *count; i++ {
		// 生成 license key
		licenseKey := cmd.generateLicenseKey(*plan)
		
		fmt.Printf("   [%d/%d] Creating: %s\n", i+1, *count, licenseKey)
		
		// 创建 license
		err := cmd.createLicense(*apiBase, token, licenseKey, *plan, 
			planConfig.ExpiryDays, planConfig.MaxDevices, planConfig.MaxIPs)
		
		if err != nil {
			fmt.Printf("        ❌ Failed: %v\n", err)
			failCount++
		} else {
			fmt.Printf("        ✅ Created successfully\n")
			successCount++
			generatedKeys = append(generatedKeys, licenseKey)
		}
	}

	// 第三步：显示结果
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Generation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Total: %d\n", *count)
	fmt.Printf("   Success: %d\n", successCount)
	fmt.Printf("   Failed: %d\n", failCount)
	fmt.Println()

	if len(generatedKeys) > 0 {
		fmt.Println("✅ Generated License Keys:")
		fmt.Println()
		for i, key := range generatedKeys {
			fmt.Printf("   %d. %s\n", i+1, key)
		}
		fmt.Println()
		fmt.Println("💡 Tip: Save these keys in a secure location")
	}

	if failCount > 0 {
		return fmt.Errorf("%d license(s) failed to create", failCount)
	}

	fmt.Println("🎉 All licenses created successfully!")
	return nil
}

// PlanConfig 定义 plan 配置
type PlanConfig struct {
	ExpiryDays int
	MaxDevices int
	MaxIPs     int
}

// getPlanConfig 获取 plan 配置
func (cmd *licenseCmd) getPlanConfig(plan string) PlanConfig {
	configs := map[string]PlanConfig{
		"free": {
			ExpiryDays: 30,
			MaxDevices: 1,
			MaxIPs:     1,
		},
		"starter": {
			ExpiryDays: 365,
			MaxDevices: 3,
			MaxIPs:     3,
		},
		"creator": {
			ExpiryDays: 365,
			MaxDevices: 5,
			MaxIPs:     5,
		},
		"pro": {
			ExpiryDays: 365,
			MaxDevices: 10,
			MaxIPs:     10,
		},
		"enterprise": {
			ExpiryDays: 36500, // 100 years
			MaxDevices: 999,
			MaxIPs:     999,
		},
	}

	return configs[plan]
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
func (cmd *licenseCmd) createLicense(apiBase, token, licenseKey, plan string, expiryDays, maxDevices, maxIPs int) error {
	createURL := fmt.Sprintf("%s/api/content?type=License", apiBase)
	
	// 构造 multipart 表单
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// 添加字段
	fields := map[string]string{
		"id":          "-1",
		"license_key": licenseKey,
		"plan":        plan,
		"expiry_days": fmt.Sprintf("%d", expiryDays),
		"max_devices": fmt.Sprintf("%d", maxDevices),
		"max_ips":     fmt.Sprintf("%d", maxIPs),
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
