package content

import (
	"errors"
	"github.com/blevesearch/bleve/mapping"
	"github.com/gofrs/uuid"
	"github.com/mdfriday/hugoverse/internal/domain/contenthub"
	"net/http"
)

type Creator func() interface{}

type Identifier interface {
	ID() string
	ContentType() string
}

type DirService interface {
	DataDir() string
	UploadDir() string
	PreviewDir() string
	PreviewFolder() string
}

type Services interface {
	SiteService
	CHService
}

type SiteService interface {
	SiteTitle() string
	BaseUrl() string
	ConfigParams() map[string]any
	DefaultTheme() string
	WorkingDir() string

	LanguageKeys() []string
	DefaultLanguage() string
	GetLanguageName(lang string) string
	GetLanguageIndex(lang string) (int, error)
	GetLanguageFolder(lang string) string
}

type CHService interface {
	WalkPages(langIndex int, walker contenthub.WalkFunc) error
	GetPageSources(page contenthub.Page) ([]contenthub.PageSource, error)
}

type Status string

const (
	Public  Status = "public"
	Pending Status = "pending"
)

type Unmarshalable interface {
	UnmarshalJSON([]byte) error
}

// Hideable lets a user keep items hidden
type Hideable interface {
	Hide(http.ResponseWriter, *http.Request) error
}

type Buildable interface {
	Build() bool
}

type Deployable interface {
	Deploy() bool
}

type RefSelectable interface {
	SelectContentTypes() []string
	SetSelectData(data map[string][][]byte)
}

// Pushable lets a user define which values of certain struct fields are
// 'pushed' down to  a client via HTTP/2 Server Push. All items in the slice
// should be the json tag names of the struct fields to which they correspond.
type Pushable interface {
	// Push the values contained by fields returned by Push must strictly be URL paths
	Push(http.ResponseWriter, *http.Request) ([]string, error)
}

// Omittable lets a user define certin fields within a content struct to remove
// from an API response. Helpful when you want data in the CMS, but not entirely
// shown or available from the content API. All items in the slice should be the
// json tag names of the struct fields to which they correspond.
type Omittable interface {
	Omit(http.ResponseWriter, *http.Request) ([]string, error)
}

// Createable accepts or rejects external POST requests to endpoints such as:
// /api/content/create?type=Review
type Createable interface {
	// Create enables external clients to submit content of a specific type
	Create(http.ResponseWriter, *http.Request) error
}

type Updateable interface {
	Update(http.ResponseWriter, *http.Request) error
}

// Trustable allows external content to be auto-approved, meaning content sent
// as an Createable will be stored in the public content bucket
type Trustable interface {
	AutoApprove(http.ResponseWriter, *http.Request) error
}

// Sortable ensures data is sortable by time
type Sortable interface {
	Time() int64
	Touch() int64
}

// CSVFormattable is implemented with the method FormatCSV, which must return the ordered
// slice of JSON struct tag names for the type implmenting it
type CSVFormattable interface {
	FormatCSV() []string
}

// Sluggable makes a struct locatable by URL with it's own path.
// As an Item implementing Sluggable, slugs may overlap. If this is an issue,
// make your content struct (or one which embeds Item) implement Sluggable
// and it will override the slug created by Item's SetSlug with your own
type Sluggable interface {
	SetSlug(string)
	ItemSlug() string
}

type Taggable interface {
	ItemTags() []string
}

type Hashable interface {
	SetHash()
	ItemHash() string
}

type Metable interface {
	SetMeta(service DirService) error
}

// Searchable ...
type Searchable interface {
	SearchMapping() (*mapping.IndexMappingImpl, error)
	IndexContent() bool
}

var (
	// ErrNoIndex is for failed checks for an index in Search map
	ErrNoIndex = errors.New("no search index found for type provided")
)

// Identifiable enables a struct to have its ID set/get. Typically this is done
// to set an ID to -1 indicating it is new for DB inserts, since by default
// a newly initialized struct would have an ID of 0, the int zero-value, and
// bbolt's starting key per bucket is 0, thus overwriting the first record.
type Identifiable interface {
	ItemID() int
	SetItemID(int)

	UniqueID() uuid.UUID
	SetUniqueID(uuid.UUID)

	String() string
	ItemName() string
}

type Statusable interface {
	ItemStatus() Status
	SetItemStatus(Status)
}

// Hookable provides our user with an easy way to intercept or add functionality
// to the different lifecycles/events a struct may encounter. Item implements
// Hookable with no-ops so our user can override only whichever ones necessary.
type Hookable interface {
	BeforeAPIResponse(http.ResponseWriter, *http.Request, []byte) ([]byte, error)
	AfterAPIResponse(http.ResponseWriter, *http.Request, []byte) error

	BeforeAPICreate(http.ResponseWriter, *http.Request) error
	AfterAPICreate(http.ResponseWriter, *http.Request) error

	BeforeAPIUpdate(http.ResponseWriter, *http.Request) error
	AfterAPIUpdate(http.ResponseWriter, *http.Request) error

	BeforeAPIDelete(http.ResponseWriter, *http.Request) error
	AfterAPIDelete(http.ResponseWriter, *http.Request) error

	BeforeAdminCreate(http.ResponseWriter, *http.Request) error
	AfterAdminCreate(http.ResponseWriter, *http.Request) error

	BeforeAdminUpdate(http.ResponseWriter, *http.Request) error
	AfterAdminUpdate(http.ResponseWriter, *http.Request) error

	BeforeAdminDelete(http.ResponseWriter, *http.Request) error
	AfterAdminDelete(http.ResponseWriter, *http.Request) error

	BeforeSave(http.ResponseWriter, *http.Request) error
	AfterSave(http.ResponseWriter, *http.Request) error

	BeforeDelete(http.ResponseWriter, *http.Request) error
	AfterDelete(http.ResponseWriter, *http.Request) error

	BeforeApprove(http.ResponseWriter, *http.Request) error
	AfterApprove(http.ResponseWriter, *http.Request) error

	BeforeReject(http.ResponseWriter, *http.Request) error
	AfterReject(http.ResponseWriter, *http.Request) error

	// Enable/Disable used for addons
	BeforeEnable(http.ResponseWriter, *http.Request) error
	AfterEnable(http.ResponseWriter, *http.Request) error

	BeforeDisable(http.ResponseWriter, *http.Request) error
	AfterDisable(http.ResponseWriter, *http.Request) error
}

const (
	typeNotRegistered = `Error:
There is no type registered for %[1]s

Add this to the file which defines %[1]s{} in the 'content' package:


	func init() {			
		item.Types["%[1]s"] = func() interface{} { return new(%[1]s) }
	}
`

	typeNotSupportedForBuilding = `Error:
There is no building feature supported for %[1]s

Site only supported at the moment.
`
)

var (
	// ErrTypeNotRegistered means content type isn't registered (not found in Types map)
	ErrTypeNotRegistered = errors.New(typeNotRegistered)

	// ErrTypeNotSupportedForBuilding means content type isn't supported for building
	ErrTypeNotSupportedForBuilding = errors.New(typeNotSupportedForBuilding)

	// ErrAllowHiddenItem should be used as an error to tell a caller of Hideable#Hide
	// that this type is hidden, but should be shown in a particular case, i.e.
	// if requested by a valid admin or user
	ErrAllowHiddenItem = errors.New(`allow hidden item`)
)
