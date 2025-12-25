package valueobject

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// LicensePlan 定义许可证套餐类型
type LicensePlan string

const (
	PlanFree       LicensePlan = "free"
	PlanStarter    LicensePlan = "starter"
	PlanCreator    LicensePlan = "creator"
	PlanPro        LicensePlan = "pro"
	PlanEnterprise LicensePlan = "enterprise"
)

// License 作为 Content ValueObject，管理用户许可证
type License struct {
	Item

	LicenseKey string      `json:"license_key"` // MDF-XXXX-XXXX-XXXX
	Plan       LicensePlan `json:"plan"`

	// 有效期
	IssueDate  int64 `json:"issue_date"`
	ExpiryDate int64 `json:"expiry_date"`

	// 激活状态
	Activated   bool  `json:"activated"`
	ActivatedAt int64 `json:"activated_at"`

	// 设备/IP 限制 (治理用)
	MaxDevices     int `json:"max_devices"`     // 最大设备数，默认 3
	MaxIPs         int `json:"max_ips"`         // 最大 IP 数，默认 3
	CurrentDevices int `json:"current_devices"` // 当前设备数
	CurrentIPs     int `json:"current_ips"`     // 当前 IP 数
}

// MarshalEditor 实现 editor.Editable 接口
func (l *License) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(l,
		editor.Field{
			View: editor.Input("LicenseKey", l, map[string]string{
				"label":       "License Key",
				"type":        "text",
				"placeholder": "MDF-XXXX-XXXX-XXXX",
			}),
		},
		editor.Field{
			View: editor.Select("Plan", l, map[string]string{
				"label": "Plan",
			}, map[string]string{
				"free":       "Free",
				"starter":    "Starter",
				"creator":    "Creator",
				"pro":        "Pro",
				"enterprise": "Enterprise",
			}),
		},
		editor.Field{
			View: editor.Input("MaxDevices", l, map[string]string{
				"label": "Max Devices",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("MaxIPs", l, map[string]string{
				"label": "Max IPs",
				"type":  "number",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render License editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (l *License) String() string {
	return fmt.Sprintf("%s (%s)", l.LicenseKey, l.Plan)
}

// SetHash 使用 LicenseKey 的 MD5 作为 hash，用于快速查找
// 存入 __contentIndex["license:{hash}"] → ID
func (l *License) SetHash() {
	l.Hash = hash.MD5(l.LicenseKey)
}

// SetSlug 使用 LicenseKey 作为 slug，存入 ns__index bucket
func (l *License) SetSlug(slug string) {
	if slug != "" {
		l.Slug = slug
	} else {
		l.Slug = l.LicenseKey
	}
}

// IndexContent 标记此类型需要被索引
func (l *License) IndexContent() bool {
	return true
}

// ========== License 转用户机制 ==========

// ToEmail 将 LicenseKey 转换为用户邮箱
// 例如: MDF-ABCD-EFGH-JKLM → abcd-efgh-jklm@mdfriday.com
func (l *License) ToEmail() string {
	key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
	return fmt.Sprintf("%s@mdfriday.com", key)
}

// ToPassword 将 LicenseKey 转换为密码 (base64 编码)
func (l *License) ToPassword() string {
	key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
	return base64.StdEncoding.EncodeToString([]byte(key))
}

// ToUserDir 生成用户目录名 (hash 前 16 位)
func (l *License) ToUserDir() string {
	return hash.MD5(l.ToEmail())[:16]
}

// IsExpired 检查许可证是否过期
func (l *License) IsExpired() bool {
	return time.Now().UnixMilli() > l.ExpiryDate
}

// IsValid 检查许可证是否有效 (已激活且未过期)
func (l *License) IsValid() bool {
	return l.Activated && !l.IsExpired()
}

// GetFeatures 获取当前套餐的功能特性
func (l *License) GetFeatures() *LicenseFeatures {
	return GetPlanFeatures(l.Plan)
}

// ========== 设备/IP 限制检查 ==========

// CanAddDevice 检查是否可以添加新设备
func (l *License) CanAddDevice() bool {
	if l.MaxDevices == -1 { // 无限制
		return true
	}
	return l.CurrentDevices < l.MaxDevices
}

// CanAddIP 检查是否可以添加新 IP
func (l *License) CanAddIP() bool {
	if l.MaxIPs == -1 { // 无限制
		return true
	}
	return l.CurrentIPs < l.MaxIPs
}

// ========== LicenseFeatures 权限定义 ==========

// LicenseFeatures 定义各套餐的功能特性
type LicenseFeatures struct {
	// 设备/IP 限制
	MaxDevices int `json:"max_devices"`
	MaxIPs     int `json:"max_ips"`

	// Sync 功能
	SyncEnabled bool `json:"sync_enabled"`
	SyncQuotaMB int  `json:"sync_quota"`

	// Publish 功能
	PublishEnabled bool `json:"publish_enabled"`
	MaxSites       int  `json:"max_sites"`
	MaxStorageMB   int  `json:"max_storage"`
	CustomDomain   bool `json:"custom_domain"`
}

// GetPlanFeatures 根据套餐类型获取功能特性
func GetPlanFeatures(plan LicensePlan) *LicenseFeatures {
	switch plan {
	case PlanFree:
		return &LicenseFeatures{
			MaxDevices:     1,
			MaxIPs:         1,
			SyncEnabled:    false,
			SyncQuotaMB:    0,
			PublishEnabled: false,
			MaxSites:       0,
			MaxStorageMB:   100,
			CustomDomain:   false,
		}
	case PlanStarter:
		return &LicenseFeatures{
			MaxDevices:     3,
			MaxIPs:         3,
			SyncEnabled:    true,
			SyncQuotaMB:    500,
			PublishEnabled: true,
			MaxSites:       3,
			MaxStorageMB:   1024,
			CustomDomain:   false,
		}
	case PlanCreator:
		return &LicenseFeatures{
			MaxDevices:     5,
			MaxIPs:         5,
			SyncEnabled:    true,
			SyncQuotaMB:    2048,
			PublishEnabled: true,
			MaxSites:       10,
			MaxStorageMB:   5120,
			CustomDomain:   true,
		}
	case PlanPro:
		return &LicenseFeatures{
			MaxDevices:     10,
			MaxIPs:         10,
			SyncEnabled:    true,
			SyncQuotaMB:    10240,
			PublishEnabled: true,
			MaxSites:       50,
			MaxStorageMB:   20480,
			CustomDomain:   true,
		}
	case PlanEnterprise:
		return &LicenseFeatures{
			MaxDevices:     -1, // 无限制
			MaxIPs:         -1, // 无限制
			SyncEnabled:    true,
			SyncQuotaMB:    51200,
			PublishEnabled: true,
			MaxSites:       -1, // 无限制
			MaxStorageMB:   102400,
			CustomDomain:   true,
		}
	default:
		return &LicenseFeatures{}
	}
}

