package valueobject

import (
	"encoding/base64"
	"fmt"
	"net/http"
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
	PlanEnjoy      LicensePlan = "enjoy"
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

	// Sync 功能
	SyncEnabled bool `json:"sync_enabled"`
	SyncQuotaMB int  `json:"sync_quota"`

	// Publish 功能
	PublishEnabled  bool `json:"publish_enabled"`
	MaxSites        int  `json:"max_sites"`
	MaxStorageMB    int  `json:"max_storage"`
	CustomDomain    bool `json:"custom_domain"`
	CustomSubDomain bool `json:"custom_sub_domain"` // 二级域名
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
				"enjoy":      "Enjoy",
				"creator":    "Creator",
				"pro":        "Pro",
				"enterprise": "Enterprise",
			}),
		},
		editor.Field{
			View: editor.Checkbox("Activated", l, map[string]string{
				"label": "Activated",
			}, map[string]string{
				"true": "yes",
			}),
		},
		editor.Field{
			View: editor.Input("IssueDate", l, map[string]string{
				"label": "Issue Date",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("ExpiryDate", l, map[string]string{
				"label": "Expiry Date",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("ActivatedAt", l, map[string]string{
				"label": "Activated At",
				"type":  "number",
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
		editor.Field{
			View: editor.Checkbox("SyncEnabled", l, map[string]string{
				"label": "Sync Enabled",
			}, map[string]string{
				"true": "sync",
			}),
		},
		editor.Field{
			View: editor.Input("SyncQuotaMB", l, map[string]string{
				"label": "Sync Quota MB",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Checkbox("PublishEnabled", l, map[string]string{
				"label": "Publish Enabled",
			}, map[string]string{
				"true": "publish",
			}),
		},
		editor.Field{
			View: editor.Input("MaxSites", l, map[string]string{
				"label": "Max Sites",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("MaxStorageMB", l, map[string]string{
				"label": "Max Storage MB",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Checkbox("CustomSubDomain", l, map[string]string{
				"label": "Custom Sub Domain",
			}, map[string]string{
				"true": "sub-domain",
			}),
		},
		editor.Field{
			View: editor.Checkbox("CustomDomain", l, map[string]string{
				"label": "Custom Domain",
			}, map[string]string{
				"true": "custom-domain",
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

func (l *License) ItemHash() string {
	return l.Hash
}

// SetSlug 使用 LicenseKey 作为 slug，存入 ns__index bucket
func (l *License) SetSlug(slug string) {
	if slug != "" {
		l.Slug = slug
	} else {
		l.Slug = l.LicenseKey
	}
}

func (l *License) ItemSlug() string {
	return l.Slug
}

// IndexContent 标记此类型需要被索引
func (l *License) IndexContent() bool {
	return true
}
func (l *License) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (l *License) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (l *License) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"license_key",
		"plan",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
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

func (l *License) ToUserShortDir() string {
	return hash.MD5(l.ToEmail())[:10]
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
	if l.CurrentDevices > l.MaxDevices {
		logger.Errorf("License %s has exceeded max devices: %d > %d", l.LicenseKey, l.CurrentDevices, l.MaxDevices)
	}

	return true
}

// CanAddIP 检查是否可以添加新 IP
func (l *License) CanAddIP() bool {
	if l.CurrentIPs > l.MaxIPs {
		logger.Errorf("License %s has exceeded max IPs: %d > %d", l.LicenseKey, l.CurrentIPs, l.MaxIPs)
	}

	return true
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
	PublishEnabled  bool `json:"publish_enabled"`
	MaxSites        int  `json:"max_sites"`
	MaxStorageMB    int  `json:"max_storage"`
	CustomDomain    bool `json:"custom_domain"`
	CustomSubDomain bool `json:"custom_sub_domain"` // 二级域名

	// 有效期（天数）
	ValidityDays int `json:"validity_days"`
}

// GetPlanFeatures 根据套餐类型获取功能特性
func GetPlanFeatures(plan LicensePlan) *LicenseFeatures {
	switch plan {
	case PlanFree:
		return &LicenseFeatures{
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
		}

	case PlanStarter:
		return &LicenseFeatures{
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
		}

	case PlanEnjoy: // ✅ 畅享版
		return &LicenseFeatures{
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
		}

	case PlanCreator:
		return &LicenseFeatures{
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
		}

	case PlanPro:
		return &LicenseFeatures{
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
		}

	case PlanEnterprise:
		return &LicenseFeatures{
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
		}
	default:
		return &LicenseFeatures{
			ValidityDays: 365, // 默认 365 天
		}
	}
}
