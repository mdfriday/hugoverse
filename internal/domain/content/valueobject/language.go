package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"net/http"
)

type Language struct {
	Item

	Name string `json:"name"`
	Code string `json:"code"`
}

// MarshalEditor writes a buffer of html to edit a Artist within the CMS
// and implements editor.Editable
func (a *Language) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(a,
		// Take note that the first argument to these Input-like functions
		// is the string version of each Artist field, and must follow
		// this pattern for auto-decoding and auto-encoding reasons:
		editor.Field{
			View: editor.Input("Name", a, map[string]string{
				"label":       "Name",
				"type":        "text",
				"placeholder": "Enter the language name here",
			}),
		},
		editor.Field{
			View: editor.Input("Code", a, map[string]string{
				"label":       "Code",
				"type":        "text",
				"placeholder": "Enter the language code here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render Artist editor view: %s", err.Error())
	}

	return view, nil
}

// String defines how a Artist is printed. Update it using more descriptive
// fields from the Artist struct type
func (a *Language) String() string {
	return a.Code
}

func (a *Language) SetHash() {
	a.Hash = hash.Fields([]string{a.Name, a.Code})
}

func (a *Language) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"name",
		"code",
	}

	for _, r := range required {
		if req.PostFormValue(r) == "" {
			err := fmt.Errorf("request missing required field: %s", r)
			return err
		}
	}

	return nil
}

func (a *Language) Approve(http.ResponseWriter, *http.Request) error {
	return nil
}

func (a *Language) IndexContent() bool {
	return true
}
