package valueobject

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/schema"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/form"
	"net/url"
)

const (
	dbBackupInfo = `
		<p class="flow-text">Database Backup Credentials:</p>
		<p>Add a user name and password to download a backup of your data via HTTP.</p>
	`
)

type Config struct {
	contentVO.Item

	Name                    string   `json:"name"`
	Domain                  string   `json:"domain"`
	Netlify                 string   `json:"netlify"`
	BindAddress             string   `json:"bind_addr"`
	HTTPPort                string   `json:"http_port"`
	HTTPSPort               string   `json:"https_port"`
	DevHTTPSPort            string   `json:"dev_https_port"`
	AdminEmail              string   `json:"admin_email"`
	ClientSecret            string   `json:"client_secret"`
	Etag                    string   `json:"etag"`
	DisableCORS             bool     `json:"cors_disabled"`
	DisableGZIP             bool     `json:"gzip_disabled"`
	DisableHTTPCache        bool     `json:"cache_disabled"`
	CacheMaxAge             int64    `json:"cache_max_age"`
	CacheInvalidate         []string `json:"cache"`
	BackupBasicAuthUser     string   `json:"backup_basic_auth_user"`
	BackupBasicAuthPassword string   `json:"backup_basic_auth_password"`

	// CouchDB 配置
	URL       string `json:"url"`        // CouchDB 服务器地址 (如 http://localhost:5984)
	AdminUser string `json:"admin_user"` // 管理员用户名
	AdminPass string `json:"admin_pass"` // 管理员密码
	DBPrefix  string `json:"db_prefix"`  // 用户数据库前缀 (如 userdb-)

	// Caddy 配置
	CaddyHost string `json:"caddy_host"` // Caddy 服务器地址 (如 localhost)
	CaddyPort string `json:"caddy_port"` // Caddy 服务器端口 (如 8080)

	// 服务器 IP 配置
	ServerIP string `json:"server_ip"` // 服务器公网 IP (用于域名检测)
}

func (c *Config) IsCacheInvalidate() bool {
	return len(c.CacheInvalidate) > 0 && c.CacheInvalidate[0] == "invalidate"
}

func (c *Config) Convert(data url.Values) (*Config, error) {
	data, err := form.Convert(data)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	dec := schema.NewDecoder()
	dec.SetAliasTag("json")     // allows simpler struct tagging when creating a content type
	dec.IgnoreUnknownKeys(true) // will skip over form values submitted, but not in struct
	err = dec.Decode(cfg, data)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Marshal() ([]byte, error) {
	j, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (c *Config) Update(key string, value any) (*Config, error) {
	kv := make(map[string]interface{})

	jsonData, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &kv)
	if err != nil {
		return nil, err
	}

	// set k/v from params to decoded map
	kv[key] = value

	data := make(url.Values)
	for k, v := range kv {
		switch v.(type) {
		case string:
			data.Set(k, v.(string))

		case []string:
			vv := v.([]string)
			for i := range vv {
				data.Add(k, vv[i])
			}

		default:
			data.Set(k, fmt.Sprintf("%v", v))
		}
	}

	return c.Convert(data)
}

// MarshalEditor writes a buffer of html to edit a Post and partially implements editor.Editable
func (c *Config) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(c,
		editor.Field{
			View: editor.Input("Name", c, map[string]string{
				"label":       "Site Name",
				"placeholder": "Add a name to this site (internal use only)",
			}),
		},
		editor.Field{
			View: editor.Input("Domain", c, map[string]string{
				"label":       "Domain Name (required for SSL certificate)",
				"placeholder": "e.g. www.example.com or example.com",
			}),
		},
		editor.Field{
			View: editor.Input("Netlify", c, map[string]string{
				"label":       "Netlify Token (required for deployment)",
				"placeholder": "e.g. nfp_Z4c2Defcv57ddXcJHd626rNQKBk9VT1rbf43",
			}),
		},
		editor.Field{
			View: editor.Input("BindAddress", c, map[string]string{
				"type": "hidden",
			}),
		},
		editor.Field{
			View: editor.Input("HTTPPort", c, map[string]string{
				"type": "hidden",
			}),
		},
		editor.Field{
			View: editor.Input("HTTPSPort", c, map[string]string{
				"type": "hidden",
			}),
		},
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
		editor.Field{
			View: editor.Input("CaddyHost", c, map[string]string{
				"label":       "Caddy Service Host",
				"type":        "text",
				"placeholder": "localhost",
			}),
		},
		editor.Field{
			View: editor.Input("CaddyPort", c, map[string]string{
				"label":       "Caddy Service Port",
				"type":        "text",
				"placeholder": "8080",
			}),
		},
		editor.Field{
			View: editor.Input("ServerIP", c, map[string]string{
				"label":       "Server Public IP (for domain DNS check)",
				"type":        "text",
				"placeholder": "1.2.3.4",
			}),
		},
		editor.Field{
			View: editor.Input("AdminEmail", c, map[string]string{
				"label": "Administrator Email (notified of internal system information)",
			}),
		},
		editor.Field{
			View: editor.Input("ClientSecret", c, map[string]string{
				"label":    "Client Secret (used to validate requests, DO NOT SHARE)",
				"disabled": "true",
			}),
		},
		editor.Field{
			View: editor.Input("ClientSecret", c, map[string]string{
				"type": "hidden",
			}),
		},
		editor.Field{
			View: editor.Input("Etag", c, map[string]string{
				"label":    "Etag Header (used to cache resources)",
				"disabled": "true",
			}),
		},
		editor.Field{
			View: editor.Input("Etag", c, map[string]string{
				"type": "hidden",
			}),
		},
		editor.Field{
			View: editor.Checkbox("DisableCORS", c, map[string]string{
				"label": "Disable CORS (so only " + c.Domain + " can fetch your data)",
			}, map[string]string{
				"true": "Disable CORS",
			}),
		},
		editor.Field{
			View: editor.Checkbox("DisableGZIP", c, map[string]string{
				"label": "Disable GZIP (will increase server speed, but also bandwidth)",
			}, map[string]string{
				"true": "Disable GZIP",
			}),
		},
		editor.Field{
			View: editor.Checkbox("DisableHTTPCache", c, map[string]string{
				"label": "Disable HTTP Cache (overrides 'Cache-Control' header)",
			}, map[string]string{
				"true": "Disable HTTP Cache",
			}),
		},
		editor.Field{
			View: editor.Input("CacheMaxAge", c, map[string]string{
				"label": "Max-Age value for HTTP caching (in seconds, 0 = 2592000)",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Checkbox("CacheInvalidate", c, map[string]string{
				"label": "Invalidate cache on save",
			}, map[string]string{
				"invalidate": "Invalidate Cache",
			}),
		},
		editor.Field{
			View: []byte(dbBackupInfo),
		},
		editor.Field{
			View: editor.Input("BackupBasicAuthUser", c, map[string]string{
				"label":       "HTTP Basic Auth User",
				"placeholder": "Enter a user name for Basic Auth access",
				"type":        "text",
			}),
		},
		editor.Field{
			View: editor.Input("BackupBasicAuthPassword", c, map[string]string{
				"label":       "HTTP Basic Auth Password",
				"placeholder": "Enter a password for Basic Auth access",
				"type":        "password",
			}),
		},
	)
	if err != nil {
		return nil, err
	}

	open := []byte(`
	<div class="card">
		<div class="card-content">
			<div class="card-title">System Configuration</div>
		</div>
		<form action="/admin/configure" method="post">
	`)
	close := []byte(`</form></div>`)
	script := []byte(`
	<script>
		$(function() {
			// hide default fields & labels unnecessary for the config
			var fields = $('.default-fields');
			fields.css('position', 'relative');
			fields.find('input:not([type=submit])').remove();
			fields.find('label').remove();
			fields.find('button').css({
				position: 'absolute',
				top: '-10px',
				right: '0px'
			});

			var contentOnly = $('.content-only.__ponzu');
			contentOnly.hide();
			contentOnly.find('input, textarea, select').attr('name', '');

			// adjust layout of td so save button is in same location as usual
			fields.find('td').css('float', 'right');

			// stop some fixed config settings from being modified
			fields.find('input[name=client_secret]').attr('name', '');
		});
	</script>
	`)

	view = append(open, view...)
	view = append(view, close...)
	view = append(view, script...)

	return view, nil
}
