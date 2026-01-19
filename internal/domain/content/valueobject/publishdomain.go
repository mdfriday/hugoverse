package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

type PublishDomain struct {
	Item

	License   string `json:"license"`    // 关联的 License Key
	Folder    string `json:"folder"`     // 关联的用户文件夹
	SubDomain string `json:"sub_domain"` // 关联的子域名
	CusDomain string `json:"cus_domain"` // 关联的自定义域名
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
			View: editor.Input("Folder", d, map[string]string{
				"label": "User Folder",
				"type":  "text",
			}),
		},
		editor.Field{
			View: editor.Input("SubDomain", d, map[string]string{
				"label":       "Sub Domain",
				"type":        "text",
				"placeholder": "blog.example.com",
			}),
		},
		editor.Field{
			View: editor.Input("CusDomain", d, map[string]string{
				"label": "Custom Domain",
				"type":  "text",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render PublishDomain editor view: %s", err.Error())
	}

	return view, nil
}

func (d *PublishDomain) String() string {
	return d.License
}

func (d *PublishDomain) SetHash() {
	d.Hash = hash.MD5(d.License)
}

func (d *PublishDomain) ItemHash() string {
	return d.Hash
}

func (d *PublishDomain) SetSlug(slug string) {
	d.Slug = d.License
}

func (d *PublishDomain) ItemSlug() string {
	return d.Slug
}

// IndexContent 标记此类型需要被索引
func (d *PublishDomain) IndexContent() bool {
	return true
}

func (d *PublishDomain) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (d *PublishDomain) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}
func (d *PublishDomain) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
	// do initial user authentication here on the request, checking for a
	// token or cookie, or that certain form fields are set and valid

	// for example, this will check if the request was made by a CMS admin user:
	//if !user.IsValid(req) {
	//	return api.ErrNoAuth
	//}

	// you could then to data validation on the request post form, or do it in
	// the Create method, which is called after BeforeAPICreate

	return nil
}
func (d *PublishDomain) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (d *PublishDomain) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"license",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}
