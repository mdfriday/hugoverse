package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// LicenseDevice 设备记录表 (治理用)
// BoltDB 存储: licensedevice / licensedevice__index
// Hash: MD5(License:DeviceID) | Slug: {License}:{DeviceID[:8]}
type LicenseDevice struct {
	Item

	License    string `json:"license"`     // 关联的 License Key
	DeviceID   string `json:"device_id"`   // 设备唯一标识
	DeviceName string `json:"device_name"` // 设备名称 (UA/OS 等)
	DeviceType string `json:"device_type"` // desktop / mobile / tablet

	// 使用信息
	FirstSeenAt int64 `json:"first_seen_at"` // 首次使用时间
	LastSeenAt  int64 `json:"last_seen_at"`  // 最后使用时间
	AccessCount int   `json:"access_count"`  // 访问次数

	// 状态
	Status string `json:"status"` // active / blocked
}

// MarshalEditor 实现 editor.Editable 接口
func (d *LicenseDevice) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(d,
		editor.Field{
			View: editor.Input("License", d, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("DeviceID", d, map[string]string{
				"label": "Device ID",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("DeviceName", d, map[string]string{
				"label": "Device Name",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Select("DeviceType", d, map[string]string{
				"label": "Device Type",
			}, map[string]string{
				"desktop": "Desktop",
				"mobile":  "Mobile",
				"tablet":  "Tablet",
			}),
		},
		editor.Field{
			View: editor.Select("Status", d, map[string]string{
				"label": "Status",
			}, map[string]string{
				"active":  "Active",
				"blocked": "Blocked",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render LicenseDevice editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (d *LicenseDevice) String() string {
	if len(d.DeviceID) >= 8 {
		return fmt.Sprintf("%s - %s", d.DeviceID[:8], d.DeviceName)
	}
	return fmt.Sprintf("%s - %s", d.DeviceID, d.DeviceName)
}

// SetHash 使用 License + DeviceID 的组合 hash
// 存入 __contentIndex["licensedevice:{hash}"] → ID
func (d *LicenseDevice) SetHash() {
	d.Hash = hash.MD5(d.License + ":" + d.DeviceID)
}

func (d *LicenseDevice) ItemHash() string {
	return d.Hash
}

// SetSlug 使用 "License:DeviceID[:8]" 格式，支持按 License 前缀查询
func (d *LicenseDevice) SetSlug(slug string) {
	if len(d.DeviceID) >= 8 {
		d.Slug = fmt.Sprintf("%s:%s", d.License, d.DeviceID[:8])
	} else {
		d.Slug = fmt.Sprintf("%s:%s", d.License, d.DeviceID)
	}
}

func (d *LicenseDevice) ItemSlug() string {
	return d.Slug
}

// IndexContent 标记此类型需要被索引
func (d *LicenseDevice) IndexContent() bool {
	return true
}

func (d *LicenseDevice) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (d *LicenseDevice) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (d *LicenseDevice) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"license",
		"device_id",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}
