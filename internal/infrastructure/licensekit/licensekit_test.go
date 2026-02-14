package licensekit_test

import (
	"strings"
	"testing"

	"github.com/mdfriday/hugoverse/internal/infrastructure/licensekit"
)

func TestGenerateLicenseKey(t *testing.T) {
	// 生成多个 key 测试
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key := licensekit.GenerateLicenseKey()

		// 验证格式
		if !strings.HasPrefix(key, "MDF-") {
			t.Errorf("License key should start with 'MDF-', got: %s", key)
		}

		parts := strings.Split(key, "-")
		if len(parts) != 4 {
			t.Errorf("License key should have 4 parts separated by '-', got: %s", key)
		}

		// 验证每个部分长度
		for i := 1; i < 4; i++ {
			if len(parts[i]) != 4 {
				t.Errorf("Each part should be 4 characters, got: %s (part %d)", parts[i], i)
			}
		}

		// 验证唯一性
		if keys[key] {
			t.Errorf("Generated duplicate key: %s", key)
		}
		keys[key] = true
	}
}

func TestLicenseKeyToEmail(t *testing.T) {
	tests := []struct {
		licenseKey string
		expected   string
	}{
		{"MDF-ABCD-EFGH-IJKL", "abcd-efgh-ijkl@mdfriday.com"},
		{"MDF-1234-5678-9ABC", "1234-5678-9abc@mdfriday.com"},
	}

	for _, tt := range tests {
		result := licensekit.LicenseKeyToEmail(tt.licenseKey)
		if result != tt.expected {
			t.Errorf("LicenseKeyToEmail(%s) = %s, want %s", tt.licenseKey, result, tt.expected)
		}
	}
}

func TestLicenseKeyToPassword(t *testing.T) {
	licenseKey := "MDF-ABCD-EFGH-IJKL"
	password := licensekit.LicenseKeyToPassword(licenseKey)

	// 密码不应该为空
	if password == "" {
		t.Error("Password should not be empty")
	}

	// 密码应该是 base64 编码
	if !strings.Contains(password, "=") && len(password)%4 != 0 {
		t.Errorf("Password should be base64 encoded, got: %s", password)
	}
}

func TestIsValidPlan(t *testing.T) {
	validPlans := []string{"free", "starter", "enjoy", "creator", "pro", "enterprise"}
	for _, plan := range validPlans {
		if !licensekit.IsValidPlan(plan) {
			t.Errorf("Plan '%s' should be valid", plan)
		}
	}

	invalidPlans := []string{"invalid", "basic", "premium", ""}
	for _, plan := range invalidPlans {
		if licensekit.IsValidPlan(plan) {
			t.Errorf("Plan '%s' should be invalid", plan)
		}
	}
}

func TestGetPlanConfig(t *testing.T) {
	// 测试 free plan
	freeConfig := licensekit.GetPlanConfig("free")
	if freeConfig.MaxDevices != 3 {
		t.Errorf("Free plan should have 3 max devices, got: %d", freeConfig.MaxDevices)
	}
	if freeConfig.ValidityDays != 3 {
		t.Errorf("Free plan should have 3 validity days, got: %d", freeConfig.ValidityDays)
	}
	if !freeConfig.PublishEnabled {
		t.Error("Free plan should have publish enabled")
	}

	// 测试 enterprise plan
	enterpriseConfig := licensekit.GetPlanConfig("enterprise")
	if enterpriseConfig.MaxDevices != 100 {
		t.Errorf("Enterprise plan should have 100 max devices, got: %d", enterpriseConfig.MaxDevices)
	}
	if enterpriseConfig.ValidityDays != 36500 {
		t.Errorf("Enterprise plan should have 36500 validity days, got: %d", enterpriseConfig.ValidityDays)
	}

	// 测试无效 plan 返回默认值（free plan）
	invalidConfig := licensekit.GetPlanConfig("invalid_plan")
	if invalidConfig.MaxDevices != 3 {
		t.Errorf("Invalid plan should return free plan config, got max devices: %d", invalidConfig.MaxDevices)
	}
}

func TestGetValidPlans(t *testing.T) {
	plans := licensekit.GetValidPlans()
	if len(plans) != 6 {
		t.Errorf("Should have 6 valid plans, got: %d", len(plans))
	}

	expectedPlans := map[string]bool{
		"free": true, "starter": true, "enjoy": true, "creator": true, "pro": true, "enterprise": true,
	}

	for _, plan := range plans {
		if !expectedPlans[plan] {
			t.Errorf("Unexpected plan in valid plans list: %s", plan)
		}
	}
}

