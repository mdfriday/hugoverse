package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"net/http"
)

type InstanceStatus string

const (
	InstanceActive    InstanceStatus = "active"
	InstanceBlocked   InstanceStatus = "blocked"   // ❌ 直接禁用
	InstanceSuspended InstanceStatus = "suspended" // ⚠️ 降级
)

type Instance struct {
	Item

	// 唯一标识
	InstanceID string `json:"instance_id"` // SHA256(machine_id + salt)
	Domain     string `json:"domain"`      // 实例域名

	// 总体统计
	TotalLicenses int `json:"total_licenses"`
	TotalTrials   int `json:"total_trials"`

	// 基本信息（用于管理）
	Version   string `json:"version"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`

	// 状态控制（核心）
	Status InstanceStatus `json:"status"` // active / blocked / suspended

	// 心跳信息
	LastSeenAt int64 `json:"last_seen_at"`
	CreatedAt  int64 `json:"created_at"`

	// 控制策略（可选扩展）
	AllowOfflineSeconds int64 `json:"allow_offline_seconds"` // 允许离线多久
}

// MarshalEditor 实现 editor.Editable 接口
func (l *Instance) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(l,
		editor.Field{
			View: editor.Input("InstanceID", l, map[string]string{
				"label":       "Instance ID",
				"type":        "text",
				"placeholder": "Enter the Instance ID here",
			}),
		},
		editor.Field{
			View: editor.Input("Domain", l, map[string]string{
				"label":       "Domain",
				"type":        "text",
				"placeholder": "example.com",
			}),
		},
		editor.Field{
			View: editor.Input("TotalLicenses", l, map[string]string{
				"label": "Total Licenses",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("TotalTrials", l, map[string]string{
				"label": "Total Trials",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("Version", l, map[string]string{
				"label": "Version",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("IPAddress", l, map[string]string{
				"label": "IP Address",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("UserAgent", l, map[string]string{
				"label": "User Agent",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Select("Status", l, map[string]string{
				"label": "Status",
			}, map[string]string{
				"active":    "InstanceActive",
				"blocked":   "InstanceBlocked",
				"suspended": "InstanceSuspended",
			}),
		},
		editor.Field{
			View: editor.Input("LastSeenAt", l, map[string]string{
				"label": "Last Seen At",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("CreatedAt", l, map[string]string{
				"label": "Created At",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("AllowOfflineSeconds", l, map[string]string{
				"label": "Allow Offline Seconds",
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
func (l *Instance) String() string {
	return fmt.Sprintf("%s", l.InstanceID)
}

// SetHash 使用 LicenseKey 的 MD5 作为 hash，用于快速查找
// 存入 __contentIndex["license:{hash}"] → ID
func (l *Instance) SetHash() {
	l.Hash = hash.MD5(l.InstanceID)
}

func (l *Instance) ItemHash() string {
	return l.Hash
}

// SetSlug 使用 LicenseKey 作为 slug，存入 ns__index bucket
func (l *Instance) SetSlug(slug string) {
	if slug != "" {
		l.Slug = slug
	} else {
		l.Slug = l.InstanceID
	}
}

func (l *Instance) ItemSlug() string {
	return l.Slug
}

// IndexContent 标记此类型需要被索引
func (l *Instance) IndexContent() bool {
	return true
}
func (l *Instance) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (l *Instance) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (l *Instance) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"instance_id",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}
