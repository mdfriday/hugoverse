package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"net/http"
)

type Counter struct {
	Item

	Kind      string `json:"kind"`
	RequestID string `json:"request_id"`
}

// MarshalEditor writes a buffer of html to edit a Song within the CMS
// and implements editor.Editable
func (s *Counter) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(s,
		editor.Field{
			View: editor.Input("Kind", s, map[string]string{
				"label":       "Kind",
				"type":        "text",
				"placeholder": "Enter the Kind here",
			}),
		},
		editor.Field{
			View: editor.Textarea("RequestID", s, map[string]string{
				"label":       "RequestID",
				"type":        "text",
				"placeholder": "Enter the Request ID here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render Song editor view: %s", err.Error())
	}

	return view, nil
}

// String defines the display name of a Song in the CMS list-view
func (s *Counter) String() string { return fmt.Sprintf("%s:%s", s.Kind, s.RequestID) }

func (s *Counter) SetHash() {
	s.Hash = hash.Fields([]string{s.RequestID})
}

func (s *Counter) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"kind",
		"request_id",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}

func (s *Counter) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
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
func (s *Counter) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (s *Counter) Approve(res http.ResponseWriter, req *http.Request) error {
	return nil
}
func (s *Counter) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (s *Counter) IndexContent() bool {
	return true
}
