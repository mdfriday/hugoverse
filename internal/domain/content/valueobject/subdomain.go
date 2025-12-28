package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"net/http"
)

type SubDomain struct {
	Item

	Sub     string `json:"sub"`
	License string `json:"license"`
}

func (d *SubDomain) Name() string {
	return d.Sub
}

// MarshalEditor writes a buffer of html to edit a Song within the CMS
// and implements editor.Editable
func (d *SubDomain) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(d,
		editor.Field{
			View: editor.Input("Sub", d, map[string]string{
				"label":       "Sub",
				"type":        "text",
				"placeholder": "Enter the sub domain here",
			}),
		},
		editor.Field{
			View: editor.Input("License", d, map[string]string{
				"label":       "License",
				"type":        "text",
				"placeholder": "Enter the License here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render SubDomain editor view: %s", err.Error())
	}

	return view, nil
}

// String defines the display name of a Song in the CMS list-view
func (d *SubDomain) String() string { return d.Name() }

func (d *SubDomain) SetHash() {
	d.Hash = hash.MD5(d.Sub)
}

func (d *SubDomain) ItemHash() string {
	return d.Hash
}

func (d *SubDomain) SetSlug(slug string) {
	d.Slug = d.Sub
}

func (d *SubDomain) ItemSlug() string {
	return d.Slug
}

// Create implements api.Createable, and allows external POST requests from clients
// to add content as long as the request contains the json tag names of the Song
// struct fields, and is multipart encoded
func (d *SubDomain) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"sub",
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

func (d *SubDomain) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
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
func (d *SubDomain) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (d *SubDomain) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (d *SubDomain) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (d *SubDomain) IndexContent() bool {
	return true
}
