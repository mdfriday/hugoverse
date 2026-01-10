package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// LicenseUsage 许可证使用统计 (治理用)
// BoltDB 存储: licenseusage / licenseusage__index
// Hash: MD5(LicenseKey) | Slug: {LicenseKey}
type LicenseUsage struct {
	Item

	LicenseKey string `json:"license_key"` // 关联的 License Key

	// 当前使用量
	CurrentIPs     int `json:"current_ips"`     // 当前 IP 使用量
	CurrentDevices int `json:"current_devices"` // 当前设备使用量

	// 磁盘使用量 (单位: mb)
	SyncDiskUsage    int64 `json:"sync_disk_usage"`    // Sync 磁盘使用量
	PublishDiskUsage int64 `json:"publish_disk_usage"` // Publish 磁盘使用量

	// 更新时间
	LastUpdatedAt int64 `json:"last_updated_at"` // 最后更新时间
}

// MarshalEditor 实现 editor.Editable 接口
func (u *LicenseUsage) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(u,
		editor.Field{
			View: editor.Input("LicenseKey", u, map[string]string{
				"label":       "License Key",
				"type":        "text",
				"placeholder": "MDF-XXXX-XXXX-XXXX",
			}),
		},
		editor.Field{
			View: editor.Input("CurrentIPs", u, map[string]string{
				"label": "Current IPs",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("CurrentDevices", u, map[string]string{
				"label": "Current Devices",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("SyncDiskUsage", u, map[string]string{
				"label": "Sync Disk Usage (mb)",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("PublishDiskUsage", u, map[string]string{
				"label": "Publish Disk Usage (mb)",
				"type":  "number",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render LicenseUsage editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (u *LicenseUsage) String() string {
	return fmt.Sprintf("%s (IPs: %d, Devices: %d)", u.LicenseKey, u.CurrentIPs, u.CurrentDevices)
}

// SetHash 使用 LicenseKey 的 MD5 作为 hash，用于快速查找
// 存入 __contentIndex["licenseusage:{hash}"] → ID
func (u *LicenseUsage) SetHash() {
	u.Hash = hash.MD5(u.LicenseKey)
}

func (u *LicenseUsage) ItemHash() string {
	return u.Hash
}

// SetSlug 使用 LicenseKey 作为 slug，存入 ns__index bucket
func (u *LicenseUsage) SetSlug(slug string) {
	if slug != "" {
		u.Slug = slug
	} else {
		u.Slug = u.LicenseKey
	}
}

func (u *LicenseUsage) ItemSlug() string {
	return u.Slug
}

// IndexContent 标记此类型需要被索引
func (u *LicenseUsage) IndexContent() bool {
	return true
}

func (u *LicenseUsage) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (u *LicenseUsage) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (u *LicenseUsage) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"license_key",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}

// ========== LicenseUsage 辅助方法 ==========

// GetSyncDiskUsageMB 获取 Sync 磁盘使用量 (MB)
func (u *LicenseUsage) GetSyncDiskUsageMB() float64 {
	return float64(u.SyncDiskUsage) / (1024 * 1024)
}

// GetPublishDiskUsageMB 获取 Publish 磁盘使用量 (MB)
func (u *LicenseUsage) GetPublishDiskUsageMB() float64 {
	return float64(u.PublishDiskUsage) / (1024 * 1024)
}

// GetTotalDiskUsage 获取总磁盘使用量 (bytes)
func (u *LicenseUsage) GetTotalDiskUsage() int64 {
	return u.SyncDiskUsage + u.PublishDiskUsage
}

// GetTotalDiskUsageMB 获取总磁盘使用量 (MB)
func (u *LicenseUsage) GetTotalDiskUsageMB() float64 {
	return float64(u.GetTotalDiskUsage()) / (1024 * 1024)
}
