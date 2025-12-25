package valueobject

import (
	"fmt"
	
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// PublishDomain 自定义域名记录
// SSL 证书由 Caddy 自动管理和续签
// BoltDB 存储: publishdomain / publishdomain__index
// Hash: MD5(License:Domain) | Slug: {License}:{Domain}
type PublishDomain struct {
	Item

	License     string `json:"license"`      // 关联的 License Key
	PublishSite string `json:"publish_site"` // 关联的站点名称
	Domain      string `json:"domain"`       // 自定义域名 (如 blog.example.com)
	TargetPath  string `json:"target_path"`  // 指向的发布目录路径

	Status    string `json:"status"`     // active / inactive
	CreatedAt int64  `json:"created_at"` // 创建时间
	UpdatedAt int64  `json:"updated_at"` // 更新时间
}

// MarshalEditor 实现 editor.Editable 接口
func (d *PublishDomain) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(d,
		editor.Field{
			View: editor.Input("License", d, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("PublishSite", d, map[string]string{
				"label": "Publish Site",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("Domain", d, map[string]string{
				"label":       "Custom Domain",
				"type":        "text",
				"placeholder": "blog.example.com",
			}),
		},
		editor.Field{
			View: editor.Input("TargetPath", d, map[string]string{
				"label": "Target Path",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Select("Status", d, map[string]string{
				"label": "Status",
			}, map[string]string{
				"active":   "Active",
				"inactive": "Inactive",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render PublishDomain editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (d *PublishDomain) String() string {
	return d.Domain
}

// SetHash 使用 License + Domain 的组合 hash
// 存入 __contentIndex["publishdomain:{hash}"] → ID
func (d *PublishDomain) SetHash() {
	d.Hash = hash.MD5(d.License + ":" + d.Domain)
}

// SetSlug 使用 "License:Domain" 格式，支持按 License 前缀查询
func (d *PublishDomain) SetSlug(slug string) {
	d.Slug = fmt.Sprintf("%s:%s", d.License, d.Domain)
}

// IndexContent 标记此类型需要被索引
func (d *PublishDomain) IndexContent() bool {
	return true
}

// IsActive 检查域名是否处于活跃状态
func (d *PublishDomain) IsActive() bool {
	return d.Status == "active"
}

// ToCaddyConfig 生成 Caddy 配置片段
// Caddy 会自动申请和续签 Let's Encrypt 证书
func (d *PublishDomain) ToCaddyConfig() string {
	return fmt.Sprintf(`%s {
    root * %s
    file_server
}`, d.Domain, d.TargetPath)
}

