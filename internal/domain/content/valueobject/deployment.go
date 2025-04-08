package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"log"
	"net/http"
	"strings"
)

type Deployment struct {
	Item

	Domain string `json:"domain"`

	SiteID   string `json:"site_id"`
	SiteName string `json:"site_name"`
	SitePath string `json:"site_path"`
	HostName string `json:"host_name"`

	Status string `json:"status"`

	refSelData map[string][][]byte
}

// MarshalEditor writes a buffer of html to edit a Song within the CMS
// and implements editor.Editable
func (s *Deployment) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(s,
		editor.Field{
			View: editor.RefSelect("Domain", s, map[string]string{
				"label": "Domain",
			},
				"Domain",
				`{{ .sub }}.{{ .root }}`,
				s.refSelData["Domain"],
			),
		},
		editor.Field{
			View: editor.Input("SiteID", s, map[string]string{
				"label":       "SiteID",
				"type":        "text",
				"placeholder": "Enter the site id here",
			}),
		},
		editor.Field{
			View: editor.Input("SiteName", s, map[string]string{
				"label":       "SiteName",
				"type":        "text",
				"placeholder": "Enter the site name here",
			}),
		},
		editor.Field{
			View: editor.Input("SitePath", s, map[string]string{
				"label":       "SitePath",
				"type":        "text",
				"placeholder": "Enter the site path here",
			}),
		},
		editor.Field{
			View: editor.Input("HostName", s, map[string]string{
				"label":       "HostName",
				"type":        "text",
				"placeholder": "Enter the host name here",
			}),
		},
		editor.Field{
			View: editor.Input("Status", s, map[string]string{
				"label":       "Status",
				"type":        "text",
				"placeholder": "Enter the site deployment status here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render Author editor view: %s", err.Error())
	}

	return view, nil
}

func (s *Deployment) SetSelectData(data map[string][][]byte) {
	s.refSelData = data
}

func (s *Deployment) SelectContentTypes() []string {
	return []string{"Domain"}
}

// String defines the display name of a Song in the CMS list-view
func (s *Deployment) String() string {
	d, _ := extractTypeAndID(s.Domain)

	return strings.Join([]string{d, s.HostName}, " - ")
}

func (s *Deployment) SetHash() {
	s.Hash = hash.Fields([]string{s.Domain, s.HostName})
}

// Create implements api.Createable, and allows external POST requests from clients
// to add content as long as the request contains the json tag names of the Song
// struct fields, and is multipart encoded
func (s *Deployment) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"domain",
		"host_name",
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
func (s *Deployment) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
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
func (s *Deployment) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	addr := req.RemoteAddr
	log.Println("AfterAPICreate: Deployment sent by:", addr, "titled:", req.PostFormValue("title"))

	return nil
}

// Approve implements editor.Mergeable, which enables content supplied by external
// clients to be approved and thus added to the public content API. Before content
// is approved, it is waiting in the Pending bucket, and can only be approved in
// the CMS if the Mergeable interface is satisfied. If not, you will not see this
// content show up in the CMS.
func (s *Deployment) Approve(res http.ResponseWriter, req *http.Request) error {
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
func (s *Deployment) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (s *Deployment) IndexContent() bool {
	return true
}

func (s *Deployment) IsNewDeployment() bool {
	return s.SiteID == ""
}
