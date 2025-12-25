package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// SyncUsage Sync 使用量记录
// BoltDB 存储: syncusage / syncusage__index
// Hash: MD5(SyncAccount:RecordedAt) | 用于记录历史使用量
type SyncUsage struct {
	Item

	SyncAccount   string `json:"sync_account"`   // 关联的 SyncAccount (License Key)
	DocumentCount int    `json:"document_count"` // 文档数量
	StorageBytes  int64  `json:"storage_bytes"`  // 存储字节数
	QuotaBytes    int64  `json:"quota_bytes"`    // 配额字节数
	LastSyncAt    int64  `json:"last_sync_at"`   // 最后同步时间
	RecordedAt    int64  `json:"recorded_at"`    // 记录时间
}

// MarshalEditor 实现 editor.Editable 接口
func (u *SyncUsage) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(u,
		editor.Field{
			View: editor.Input("SyncAccount", u, map[string]string{
				"label": "Sync Account",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("DocumentCount", u, map[string]string{
				"label": "Document Count",
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
			View: editor.Input("QuotaBytes", u, map[string]string{
				"label": "Quota (bytes)",
				"type":  "number",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render SyncUsage editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (u *SyncUsage) String() string {
	return fmt.Sprintf("%s: %d docs, %d bytes", u.SyncAccount, u.DocumentCount, u.StorageBytes)
}

// SetHash 使用 SyncAccount + RecordedAt 的组合 hash
// 存入 __contentIndex["syncusage:{hash}"] → ID
func (u *SyncUsage) SetHash() {
	u.Hash = hash.MD5(fmt.Sprintf("%s:%d", u.SyncAccount, u.RecordedAt))
}

// SetSlug 使用 "SyncAccount:RecordedAt" 格式
func (u *SyncUsage) SetSlug(req *http.Request) {
	u.Slug = fmt.Sprintf("%s:%d", u.SyncAccount, u.RecordedAt)
}

// IndexContent 标记此类型需要被索引
func (u *SyncUsage) IndexContent() bool {
	return true
}

// UsagePercentage 计算使用百分比
func (u *SyncUsage) UsagePercentage() float64 {
	if u.QuotaBytes == 0 {
		return 0
	}
	return float64(u.StorageBytes) / float64(u.QuotaBytes) * 100
}

// IsOverQuota 检查是否超出配额
func (u *SyncUsage) IsOverQuota() bool {
	return u.StorageBytes > u.QuotaBytes
}

