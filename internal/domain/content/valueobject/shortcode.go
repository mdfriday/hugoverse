package valueobject

import (
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/pkg/editor"
	"github.com/mdfriday/hugoverse/pkg/images"
	"net/http"
)

type ShortCode struct {
	Item

	Name     string   `json:"name"`
	Desc     string   `json:"desc"`
	Template string   `json:"template"`
	Example  string   `json:"example"`
	Tags     []string `json:"tags"`

	Asset   string `json:"asset"`
	AssetID string `json:"asset_id"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// MarshalEditor writes a buffer of html to edit a Song within the CMS
// and implements editor.Editable
func (s *ShortCode) MarshalEditor() ([]byte, error) {
	view, err := editor.Form(s,
		editor.Field{
			View: editor.Input("Name", s, map[string]string{
				"label":       "Name",
				"type":        "text",
				"placeholder": "Enter the name here",
			}),
		},
		editor.Field{
			View: editor.Textarea("Desc", s, map[string]string{
				"label":       "Description",
				"type":        "textarea",
				"placeholder": "Enter the description here",
			}),
		},
		editor.Field{
			View: editor.Textarea("Template", s, map[string]string{
				"label":       "Template",
				"type":        "textarea",
				"placeholder": "Enter the Template here",
			}),
		},
		editor.Field{
			View: editor.Textarea("Example", s, map[string]string{
				"label":       "Example",
				"type":        "textarea",
				"placeholder": "Enter the Example here",
			}),
		},
		editor.Field{
			View: editor.File("Asset", s, map[string]string{
				"label":       "Asset",
				"placeholder": "Upload the asset here",
			}),
		},
		editor.Field{
			View: editor.Input("AssetID", s, map[string]string{
				"label":       "AssetID",
				"type":        "text",
				"placeholder": "Enter the asset id here",
			}),
		},
		editor.Field{
			View: editor.Input("Width", s, map[string]string{
				"label":       "Width",
				"type":        "text",
				"placeholder": "Enter the width here",
			}),
		},
		editor.Field{
			View: editor.Input("Height", s, map[string]string{
				"label":       "Height",
				"type":        "text",
				"placeholder": "Enter the height here",
			}),
		},
		editor.Field{
			View: editor.Tags("Tags", s, map[string]string{
				"label":       "Tags",
				"placeholder": "Upload the tags here",
			}),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to render Image editor view: %s", err.Error())
	}

	return view, nil
}

// String defines the display name of a Song in the CMS list-view
func (s *ShortCode) String() string { return s.Name }

// Create implements api.Createable, and allows external POST requests from clients
// to add content as long as the request contains the json tag names of the Song
// struct fields, and is multipart encoded
func (s *ShortCode) Create(res http.ResponseWriter, req *http.Request) error {
	// do form data validation for required fields
	required := []string{
		"name",
		"template",
		"example",
		"asset",
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
func (s *ShortCode) BeforeAPICreate(res http.ResponseWriter, req *http.Request) error {
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
func (s *ShortCode) AfterAPICreate(res http.ResponseWriter, req *http.Request) error {
	return nil
}

// Approve implements editor.Mergeable, which enables content supplied by external
// clients to be approved and thus added to the public content API. Before content
// is approved, it is waiting in the Pending bucket, and can only be approved in
// the CMS if the Mergeable interface is satisfied. If not, you will not see this
// content show up in the CMS.
func (s *ShortCode) Approve(res http.ResponseWriter, req *http.Request) error {
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
func (s *ShortCode) AutoApprove(res http.ResponseWriter, req *http.Request) error {
	// Use AutoApprove to check for trust-specific headers or whitelisted IPs,
	// etc. Remember, you will not be able to Approve or Reject content that
	// is auto-approved. You could add a field to Song, i.e.
	// AutoApproved bool `json:auto_approved`
	// and set that data here, as it is called before the content is saved, but
	// after the BeforeSave hook.

	return nil
}

func (s *ShortCode) IndexContent() bool {
	return true
}

// SearchMapping returns a default implementation of a Bleve IndexMappingImpl
// partially implements search.Searchable
//func (s *Image) SearchMapping() (*mapping.IndexMappingImpl, error) {
//	indexMapping := bleve.NewIndexMapping()
//
//	// 定义文档 Mapping
//	imageMapping := bleve.NewDocumentMapping()
//
//	// 定义 Tags 字段为 text 类型，并使用标准分词器
//	tagsFieldMapping := bleve.NewTextFieldMapping()
//	tagsFieldMapping.Analyzer = "standard" // 使用标准分词器
//	imageMapping.AddFieldMappingsAt("Tags", tagsFieldMapping)
//
//	// 绑定 ImageDocument 到索引
//	indexMapping.AddDocumentMapping("image", imageMapping)
//
//	return indexMapping, nil
//}

func (s *ShortCode) SetMeta(service content.DirService) error {
	if s.Asset == "" {
		return nil
	}

	absPath, err := getAssetAbsPath(s.Asset, service.UploadDir())
	if err != nil {
		return fmt.Errorf("error getting absolute path: %v", err)
	}

	width, height, err := images.GetImageDimensions(absPath)
	if err != nil {
		return fmt.Errorf("error getting image dimensions: %v", err)

	}
	s.Width = width
	s.Height = height

	return nil
}

func (s *ShortCode) ItemTags() []string {
	return s.Tags
}
