package valueobject

import (
	"bytes"
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"net/http"
	"text/template"
)

type Post struct {
	Item

	Title   string   `json:"title"`
	Content string   `json:"content"`
	Author  string   `json:"author"`
	Params  string   `json:"params"`
	Assets  []string `json:"assets"`
}

// MarshalEditor writes a buffer of html to edit a Song within the CMS
// and implements editor.Editable
func (s *Post) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(s,
		editor.Field{
			View: editor.Input("Title", s, map[string]string{
				"label":       "Title",
				"type":        "text",
				"placeholder": "Enter the Title here",
			}),
		},
		editor.Field{
			View: editor.Textarea("Content", s, map[string]string{
				"label":       "Content",
				"type":        "textarea",
				"placeholder": "Enter the Content in markdown here",
			}),
		},
		editor.Field{
			View: editor.Input("Author", s, map[string]string{
				"label":       "Author",
				"type":        "text",
				"placeholder": "Enter the Author name here",
			}),
		},
		editor.Field{
			View: editor.Textarea("Params", s, map[string]string{
				"label":       "Params",
				"type":        "textarea",
				"placeholder": "Enter the params here in yaml format",
			}),
		},
		editor.Field{
			View: editor.FileRepeater("Assets", s, map[string]string{
				"label":       "Assets",
				"placeholder": "Upload the Assets here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render Song editor view: %s", err.Error())
	}

	return view, nil
}

// String defines the display name of a Song in the CMS list-view
func (s *Post) String() string { return s.Title }

func (s *Post) SetHash() {
	s.Hash = hash.Fields([]string{s.Content})
}

// Create implements api.Createable, and allows external POST requests from clients
// to add content as long as the request contains the json tag names of the Song
// struct fields, and is multipart encoded
func (s *Post) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"content",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}

// BeforeAPICreate is only called if the Song type implements api.Createable
// It is called before Create, and returning an error will cancel the request
// causing the system to reject the data sent in the POST
func (s *Post) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
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

// AfterAPICreate is called after Create, and is useful for logging or triggering
// notifications, etc. after the data is saved to the database, etc.
// The request has a context containing the databse 'target' affected by the
// request. Ex. Song__pending:3 or Song:8 depending if Song implements api.Trustable
func (s *Post) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}

// Approve implements editor.Mergeable, which enables content supplied by external
// clients to be approved and thus added to the public content API. Before content
// is approved, it is waiting in the Pending bucket, and can only be approved in
// the CMS if the Mergeable interface is satisfied. If not, you will not see this
// content show up in the CMS.
func (s *Post) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}

/*
   NOTICE: if AutoApprove (seen below) is implemented, the Approve method above will have no
   effect, except to add the Public / Pending toggle in the CMS UI. Though, no
   Song content would be in Pending, since all externally submitting Song data
   is immediately approved.
*/

// AutoApprove implements api.Trustable, and will automatically approve content
// that has been submitted by an external client via api.Createable. Be careful
// when using AutoApprove, because content will immediately be available through
// your public content API. If the Trustable interface is satisfied, the AfterApprove
// method is bypassed. The
func (s *Post) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (s *Post) IndexContent() bool {
	return true
}

func (s *Post) Push(http.ResponseWriter, *http.Request) ([]string, error) {
	return []string{"author"}, nil
}

func (s *Post) FrontMatter() string {
	return fmt.Sprintf("---\ntitle: %s\n---", s.Title)
}

func (s *Post) FullContent() string {
	return s.FrontMatter() + "\n" + s.Content
}

func (s *Post) Markdown() ([]byte, error) {
	const postTemplate = `
{{.Content}}
`
	tmpl, err := template.New("toml").Parse(postTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse toml template error : %v", err)
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, s); err != nil {
		return nil, fmt.Errorf("execute toml template error: %v", err)
	}

	return result.Bytes(), nil
}
