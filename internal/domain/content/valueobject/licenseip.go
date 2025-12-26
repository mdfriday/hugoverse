package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// LicenseIP IP 记录表 (治理用)
// BoltDB 存储: licenseip / licenseip__index
// Hash: MD5(License:IPAddress) | Slug: {License}:{IPAddress}
type LicenseIP struct {
	Item

	License   string `json:"license"`    // 关联的 License Key
	IPAddress string `json:"ip_address"` // IP 地址

	// 地理位置信息 (可选)
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`

	// 使用信息
	FirstSeenAt int64 `json:"first_seen_at"` // 首次使用时间
	LastSeenAt  int64 `json:"last_seen_at"`  // 最后使用时间
	AccessCount int   `json:"access_count"`  // 访问次数

	// 状态
	Status string `json:"status"` // active / blocked
}

// MarshalEditor 实现 editor.Editable 接口
func (i *LicenseIP) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(i,
		editor.Field{
			View: editor.Input("License", i, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("IPAddress", i, map[string]string{
				"label": "IP Address",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("Country", i, map[string]string{
				"label": "Country",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("Region", i, map[string]string{
				"label": "Region",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("City", i, map[string]string{
				"label": "City",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Select("Status", i, map[string]string{
				"label": "Status",
			}, map[string]string{
				"active":  "Active",
				"blocked": "Blocked",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render LicenseIP editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (i *LicenseIP) String() string {
	if i.Country != "" {
		return fmt.Sprintf("%s (%s)", i.IPAddress, i.Country)
	}
	return i.IPAddress
}

// SetHash 使用 License + IPAddress 的组合 hash
// 存入 __contentIndex["licenseip:{hash}"] → ID
func (i *LicenseIP) SetHash() {
	i.Hash = hash.MD5(i.License + ":" + i.IPAddress)
}

func (i *LicenseIP) ItemHash() string {
	return i.Hash
}

// SetSlug 使用 "License:IPAddress" 格式，支持按 License 前缀查询
func (i *LicenseIP) SetSlug(slug string) {
	i.Slug = fmt.Sprintf("%s:%s", i.License, i.IPAddress)
}

func (i *LicenseIP) ItemSlug() string {
	return i.Slug
}

// IndexContent 标记此类型需要被索引
func (i *LicenseIP) IndexContent() bool {
	return true
}

func (i *LicenseIP) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (i *LicenseIP) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (i *LicenseIP) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"license",
		"ip_address",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}
