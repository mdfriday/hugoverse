package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// SyncAccount License 对应的 CouchDB 账号
// BoltDB 存储: syncaccount / syncaccount__index
// Hash: MD5(License) | Slug: {License}
type SyncAccount struct {
	Item

	License    string `json:"license"`     // 关联的 License Key
	Email      string `json:"email"`       // CouchDB 用户邮箱
	DBName     string `json:"db_name"`     // CouchDB 数据库名
	DBEndpoint string `json:"db_endpoint"` // CouchDB 数据库端点 URL
	Status     string `json:"status"`      // active / suspended
	CreatedAt  int64  `json:"created_at"`  // 创建时间
}

// MarshalEditor 实现 editor.Editable 接口
func (s *SyncAccount) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(s,
		editor.Field{
			View: editor.Input("License", s, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("Email", s, map[string]string{
				"label": "Email",
				"type":  "email",
			}),
		},
		editor.Field{
			View: editor.Input("DBName", s, map[string]string{
				"label": "Database Name",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("DBEndpoint", s, map[string]string{
				"label": "Database Endpoint",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Select("Status", s, map[string]string{
				"label": "Status",
			}, map[string]string{
				"active":    "Active",
				"suspended": "Suspended",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render SyncAccount editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (s *SyncAccount) String() string {
	return fmt.Sprintf("%s - %s", s.Email, s.DBName)
}

// SetHash 使用 License 的 hash，一个 License 对应一个 SyncAccount
// 存入 __contentIndex["syncaccount:{hash}"] → ID
func (s *SyncAccount) SetHash() {
	s.Hash = hash.MD5(s.License)
}

// SetSlug 使用 License 作为 slug
func (s *SyncAccount) SetSlug(req *http.Request) {
	s.Slug = s.License
}

// IndexContent 标记此类型需要被索引
func (s *SyncAccount) IndexContent() bool {
	return true
}

