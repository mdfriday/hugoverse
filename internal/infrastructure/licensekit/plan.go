package licensekit

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

// GetPlanConfig 获取 plan 配置
func GetPlanConfig(plan string) PlanConfig {
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
			ValidityDays:    7, // ✅ 7 天
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

	config, exists := configs[plan]
	if !exists {
		// 返回默认的 free plan 配置
		return configs["free"]
	}

	return config
}

// IsValidPlan 检查 plan 是否有效
func IsValidPlan(plan string) bool {
	validPlans := map[string]bool{
		"free": true, "starter": true, "enjoy": true, "creator": true, "pro": true, "enterprise": true,
	}
	return validPlans[plan]
}

// GetValidPlans 获取所有有效的 plan 列表
func GetValidPlans() []string {
	return []string{"free", "starter", "enjoy", "creator", "pro", "enterprise"}
}
