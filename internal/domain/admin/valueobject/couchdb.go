package valueobject

import (
	"fmt"

	"github.com/mdfriday/hugoverse/pkg/editor"
)

// CouchDBConfig CouchDB 管理员配置
// 用于创建用户数据库和用户账号
type CouchDBConfig struct {
	URL      string `json:"url"`       // CouchDB 服务器地址 (如 http://localhost:5984)
	AdminUser string `json:"admin_user"` // 管理员用户名
	AdminPass string `json:"admin_pass"` // 管理员密码
	DBPrefix  string `json:"db_prefix"`  // 用户数据库前缀 (如 userdb-)
}

// MarshalEditor 实现 editor.Editable 接口
func (c *CouchDBConfig) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(c,
		editor.Field{
			View: editor.Input("URL", c, map[string]string{
				"label":       "CouchDB URL",
				"type":        "text",
				"placeholder": "http://localhost:5984",
			}),
		},
		editor.Field{
			View: editor.Input("AdminUser", c, map[string]string{
				"label":       "Admin Username",
				"type":        "text",
				"placeholder": "admin",
			}),
		},
		editor.Field{
			View: editor.Input("AdminPass", c, map[string]string{
				"label": "Admin Password",
				"type":  "password",
			}),
		},
		editor.Field{
			View: editor.Input("DBPrefix", c, map[string]string{
				"label":       "Database Prefix",
				"type":        "text",
				"placeholder": "userdb-",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render CouchDBConfig editor view: %s", err.Error())
	}

	return view, nil
}

// String 返回配置的描述
func (c *CouchDBConfig) String() string {
	return fmt.Sprintf("CouchDB: %s (prefix: %s)", c.URL, c.DBPrefix)
}

// DefaultCouchDBConfig 返回默认配置
func DefaultCouchDBConfig() *CouchDBConfig {
	return &CouchDBConfig{
		URL:       "http://localhost:5984",
		AdminUser: "admin",
		AdminPass: "",
		DBPrefix:  "userdb-",
	}
}

