package valueobject

import (
	"fmt"
	"net/http"

	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// LicenseTrial 记录试用 License 申请信息
// 一个邮箱只能申请一次试用码
type LicenseTrial struct {
	Item

	Email   string `json:"email"`   // 申请邮箱（唯一标识）
	License string `json:"license"` // 生成的 License Key
}

func (t *LicenseTrial) Name() string {
	return t.Email
}

// MarshalEditor writes a buffer of html to edit a LicenseTrial within the CMS
// and implements editor.Editable
func (t *LicenseTrial) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(t,
		editor.Field{
			View: editor.Input("Email", t, map[string]string{
				"label":       "Email",
				"type":        "email",
				"placeholder": "Enter the email here",
			}),
		},
		editor.Field{
			View: editor.Input("License", t, map[string]string{
				"label":       "License",
				"type":        "text",
				"placeholder": "Enter the License Key here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render LicenseTrial editor view: %s", err.Error())
	}

	return view, nil
}

// String defines the display name of a LicenseTrial in the CMS list-view
func (t *LicenseTrial) String() string { return t.Name() }

// SetHash 使用 email 作为 hash key
func (t *LicenseTrial) SetHash() {
	t.Hash = hash.MD5(t.Email)
}

func (t *LicenseTrial) ItemHash() string {
	return t.Hash
}

// SetSlug 使用 email 作为 slug
func (t *LicenseTrial) SetSlug(slug string) {
	t.Slug = t.Email
}

func (t *LicenseTrial) ItemSlug() string {
	return t.Slug
}

// Create implements api.Createable, and allows external POST requests from clients
// to add content as long as the request contains the json tag names of the LicenseTrial
// struct fields, and is multipart encoded
func (t *LicenseTrial) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"email",
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

func (t *LicenseTrial) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (t *LicenseTrial) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (t *LicenseTrial) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (t *LicenseTrial) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	return nil
}

func (t *LicenseTrial) IndexContent() bool {
	return true
}

