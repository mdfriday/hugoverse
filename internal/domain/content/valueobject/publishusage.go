package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// PublishUsage Publish 容量记录
// BoltDB 存储: publishusage / publishusage__index
// Hash: MD5(License:RecordedAt) | 用于记录历史容量
type PublishUsage struct {
	Item

	License      string `json:"license"`       // 关联的 License Key
	SiteCount    int    `json:"site_count"`    // 站点数量
	StorageBytes int64  `json:"storage_bytes"` // 存储字节数
	MaxSites     int    `json:"max_sites"`     // 最大站点数
	QuotaBytes   int64  `json:"quota_bytes"`   // 配额字节数
	RecordedAt   int64  `json:"recorded_at"`   // 记录时间
}

// MarshalEditor 实现 editor.Editable 接口
func (u *PublishUsage) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(u,
		editor.Field{
			View: editor.Input("License", u, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("SiteCount", u, map[string]string{
				"label": "Site Count",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("StorageBytes", u, map[string]string{
				"label": "Storage (bytes)",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("MaxSites", u, map[string]string{
				"label": "Max Sites",
				"type":  "number",
			}),
		},
		editor.Field{
			View: editor.Input("QuotaBytes", u, map[string]string{
				"label": "Quota (bytes)",
				"type":  "number",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render PublishUsage editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (u *PublishUsage) String() string {
	return fmt.Sprintf("%s: %d sites, %d bytes", u.License, u.SiteCount, u.StorageBytes)
}

// SetHash 使用 License + RecordedAt 的组合 hash
// 存入 __contentIndex["publishusage:{hash}"] → ID
func (u *PublishUsage) SetHash() {
	u.Hash = hash.MD5(fmt.Sprintf("%s:%d", u.License, u.RecordedAt))
}

// SetSlug 使用 "License:RecordedAt" 格式
func (u *PublishUsage) SetSlug(req *http.Request) {
	u.Slug = fmt.Sprintf("%s:%d", u.License, u.RecordedAt)
}

// IndexContent 标记此类型需要被索引
func (u *PublishUsage) IndexContent() bool {
	return true
}

// StoragePercentage 计算存储使用百分比
func (u *PublishUsage) StoragePercentage() float64 {
	if u.QuotaBytes == 0 {
		return 0
	}
	return float64(u.StorageBytes) / float64(u.QuotaBytes) * 100
}

// IsStorageOverQuota 检查存储是否超出配额
func (u *PublishUsage) IsStorageOverQuota() bool {
	return u.StorageBytes > u.QuotaBytes
}

// CanAddSite 检查是否可以添加新站点
func (u *PublishUsage) CanAddSite() bool {
	if u.MaxSites == -1 { // 无限制
		return true
	}
	return u.SiteCount < u.MaxSites
}

