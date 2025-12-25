package valueobject

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// PublishSite 发布站点记录
// BoltDB 存储: publishsite / publishsite__index
// Hash: MD5(License:Name) | Slug: {License}:{Name}
type PublishSite struct {
	Item

	License    string `json:"license"`     // 关联的 License Key
	Name       string `json:"name"`        // 站点名称
	SiteType   string `json:"site_type"`   // site / article
	Asset      string `json:"asset"`       // 资源文件路径 (zip)
	Size       int64  `json:"size"`        // 大小 (bytes)
	FolderPath string `json:"folder_path"` // 发布目录路径
	PublicURL  string `json:"public_url"`  // 公开访问 URL
	Status     string `json:"status"`      // pending / active / deleted
	CreatedAt  int64  `json:"created_at"`  // 创建时间
}

// MarshalEditor 实现 editor.Editable 接口
func (p *PublishSite) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(p,
		editor.Field{
			View: editor.Input("License", p, map[string]string{
				"label": "License Key",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("Name", p, map[string]string{
				"label":       "Site Name",
				"type":        "text",
				"placeholder": "my-blog",
			}),
		},
		editor.Field{
			View: editor.Select("SiteType", p, map[string]string{
				"label": "Site Type",
			}, map[string]string{
				"site":    "Site",
				"article": "Article",
			}),
		},
		editor.Field{
			View: editor.File("Asset", p, map[string]string{
				"label": "Asset (ZIP)",
			}),
		},
		editor.Field{
			View: editor.Select("Status", p, map[string]string{
				"label": "Status",
			}, map[string]string{
				"pending": "Pending",
				"active":  "Active",
				"deleted": "Deleted",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render PublishSite editor view: %s", err.Error())
	}

	return view, nil
}

// String 定义在 CMS 列表中的显示名称
func (p *PublishSite) String() string {
	return p.Name
}

// SetHash 使用 License + Name 的组合 hash
// 存入 __contentIndex["publishsite:{hash}"] → ID
func (p *PublishSite) SetHash() {
	p.Hash = hash.MD5(p.License + ":" + p.Name)
}

// SetSlug 使用 "License:Name" 格式，支持按 License 前缀查询
func (p *PublishSite) SetSlug(req *http.Request) {
	p.Slug = fmt.Sprintf("%s:%s", p.License, p.Name)
}

// IndexContent 标记此类型需要被索引
func (p *PublishSite) IndexContent() bool {
	return true
}

// Deploy 标记此类型支持部署
func (p *PublishSite) Deploy() bool {
	return true
}

// AbsAssetPath 获取资源文件的绝对路径
func (p *PublishSite) AbsAssetPath(uploadDir string) (string, error) {
	if p.Asset == "" {
		return "", fmt.Errorf("asset path is empty")
	}
	return filepath.Join(uploadDir, p.Asset), nil
}

// IsActive 检查站点是否处于活跃状态
func (p *PublishSite) IsActive() bool {
	return p.Status == "active"
}

