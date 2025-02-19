package contenthub

import (
	"bytes"
	"context"
	"github.com/mdfriday/hugoverse/internal/domain/fs"
	"github.com/mdfriday/hugoverse/internal/domain/markdown"
	"github.com/mdfriday/hugoverse/internal/domain/template"
	"github.com/mdfriday/hugoverse/pkg/cache/stale"
	"github.com/mdfriday/hugoverse/pkg/identity"
	pio "github.com/mdfriday/hugoverse/pkg/io"
	"github.com/mdfriday/hugoverse/pkg/maps"
	"github.com/mdfriday/hugoverse/pkg/media"
	"github.com/mdfriday/hugoverse/pkg/output"
	"github.com/mdfriday/hugoverse/pkg/paths"
	"github.com/mdfriday/hugoverse/pkg/text"
	"github.com/spf13/afero"
	goTmpl "html/template"
	"io"
	"time"
)

type ContentHub interface {
	CollectPages() error
	PreparePages() error
	RenderPages(td TemplateDescriptor, cb func(info PageInfo) error) error

	RenderString(ctx context.Context, args ...any) (goTmpl.HTML, error)

	SetTemplateExecutor(exec Template)

	Title
}

type Title interface {
	CreateTitle(raw string) string
}

type TemplateDescriptor interface {
	Name() string
	Extension() string
}

type PageInfo interface {
	Kind() string
	Sections() []string
	Dir() string
	Name() string
	Buffer() *bytes.Buffer
}

const (
	KindPage     = "page"
	KindHome     = "home"
	KindSection  = "section"
	KindTerm     = "term"
	KindTaxonomy = "taxonomy"
)

type Services interface {
	LangService
	FsService
	TaxonomyService
	MediaService
}

type MediaService interface {
	MediaTypes() media.Types
}

type FsService interface {
	NewFileMetaInfo(filename string) fs.FileMetaInfo
	NewFileMetaInfoWithContent(content string) fs.FileMetaInfo

	LayoutFs() afero.Fs
	ContentFs() afero.Fs

	WalkContent(start string, cb fs.WalkCallback, conf fs.WalkwayConfig) error
	WalkI18n(start string, cb fs.WalkCallback, conf fs.WalkwayConfig) error
	ReverseLookupContent(filename string, checkExists bool) ([]fs.ComponentPath, error)
}

type LangService interface {
	IsLanguageValid(lang string) bool
	GetSourceLang(source string) (string, bool)
	GetLanguageIndex(lang string) (int, error)
	GetLanguageByIndex(idx int) string
	DefaultLanguage() string
	LanguageIndexes() []int
}

type Taxonomy interface {
	Singular() string // e.g. "category"
	Plural() string   // e.g. "categories"
}

type TaxonomyService interface {
	Views() []Taxonomy
}

type Template interface {
	ExecuteWithContext(ctx context.Context, tmpl template.Preparer, wr io.Writer, data any) error
	LookupVariants(name string) []template.Preparer
	LookupVariant(name string, variants template.Variants) (template.Preparer, bool, bool)
}

type BuildStateReseter interface {
	ResetBuildState()
}

type PageIdentity interface {
	identity.Identity

	PageLanguage() string
	PageLanguageIndex() int
}

type PageSource interface {
	PageIdentity() PageIdentity
	PageFile() File

	stale.Staler

	Section() string
	Paths() *paths.Path
	Path() string
	Opener() pio.OpenReadSeekCloser
}

type ContentNode interface {
	identity.IdentityProvider
	identity.ForEeachIdentityProvider
	stale.Marker
	BuildStateReseter

	Path() string
	IsContentNodeBranch() bool
}

type WeightedContentNode interface {
	ContentNode
	Weight() int
}

type ContentConvertProvider interface {
	GetContentConvertProvider(name string) ConverterProvider
}

type ConverterRegistry interface {
	Get(name string) ConverterProvider
}

// ConverterProvider creates converters.
type ConverterProvider interface {
	New(ctx markdown.DocumentContext) (Converter, error)
	Name() string
}

// ProviderProvider creates converter providers.
type ProviderProvider interface {
	New() (ConverterProvider, error)
}

// Converter wraps the Convert method that converts some markup into
// another format, e.g. Markdown to HTML.
type Converter interface {
	Convert(ctx markdown.RenderContext) (markdown.Result, error)
}

// ContentProvider provides the content related values for a Page.
type ContentProvider interface {
	Content() (any, error)
}

// FileProvider provides the source file.
type FileProvider interface {
	File() File
}

// File represents a source file.
// This is a temporary construct until we resolve page.Page conflicts.
type File interface {
	fileOverlap
	FileWithoutOverlap
}

// Temporary to solve duplicate/deprecated names in page.Page
type fileOverlap interface {
	// Paths gets the relative path including file name and extension.
	// The directory is relative to the content root.
	RelPath() string

	// Section is first directory below the content root.
	// For page bundles in root, the Section will be empty.
	Section() string

	IsZero() bool
}

type FileWithoutOverlap interface {

	// Filename gets the full path and filename to the file.
	Filename() string

	// Dir gets the name of the directory that contains this file.
	// The directory is relative to the content root.
	Dir() string

	// Ext gets the file extension, i.e "myblogpost.md" will return "md".
	Ext() string

	// LogicalName is filename and extension of the file.
	LogicalName() string

	// BaseFileName is a filename without extension.
	BaseFileName() string

	// TranslationBaseName is a filename with no extension,
	// not even the optional language extension part.
	TranslationBaseName() string

	// ContentBaseName is a either TranslationBaseName or name of containing folder
	// if file is a leaf bundle.
	ContentBaseName() string

	// UniqueID is the MD5 hash of the file's path and is for most practical applications,
	// Hugo content files being one of them, considered to be unique.
	UniqueID() string

	FileInfo() fs.FileMetaInfo
}

type PageGroups []PageGroup

func (p PageGroups) Len() int { return len(p) }

type PageGroup interface {
	Key() string
	Pages() Pages
	Append(page Page) Pages
}

type Pages []Page

func (p Pages) Len() int { return len(p) }

// Document is the interface an indexable document in Hugo must fulfill.
type Document interface {
	// RelatedKeywords returns a list of keywords for the given index config.
	RelatedKeywords(cfg IndexConfig) ([]Keyword, error)

	// When this document was or will be published.
	PublishDate() time.Time

	// Name is used as an tiebreaker if both Weight and PublishDate are
	// the same.
	Name() string
}

// Keyword is the interface a keyword in the search index must implement.
type Keyword interface {
	String() string
}

// IndicesConfig holds a set of index configurations.
type IndicesConfig []IndexConfig

// IndexConfig configures an index.
type IndexConfig interface {
	Name() string
	Type() string
	ApplyFilter() bool

	// Pattern Contextual pattern used to convert the Param value into a string.
	// Currently only used for dates. Can be used to, say, bump posts in the same
	// time frame when searching for related documents.
	// For dates it follows Go's time.Format patterns, i.e.
	// "2006" for YYYY and "200601" for YYYYMM.
	Pattern() string

	// Weight This field's weight when doing multi-index searches. Higher is "better".
	Weight() int

	// CardinalityThreshold A percentage (0-100) used to remove common keywords from the index.
	// As an example, setting this to 50 will remove all keywords that are
	// used in more than 50% of the documents in the index.
	CardinalityThreshold() int

	// ToLower Will lower case all string values in and queries tothis index.
	// May get better accurate results, but at a slight performance cost.
	ToLower() bool

	ToKeywords(v any) ([]Keyword, error)
}

type PageWrapper interface {
	UnwrapPage() Page
}

// PageContext provides contextual information about this page, for error
// logging and similar.
type PageContext interface {
	PosOffset(offset int) text.Position
}

// Page is the core interface in Hugo.
type Page interface {
	RawContentProvider

	PageSource
	PageMeta
	PagerManager

	Name() string
	Title() string
	Kind() string
	IsHome() bool
	IsPage() bool
	IsSection() bool
	IsAncestor(other Page) bool
	Eq(other Page) bool

	Layouts() []string
	PageOutputs() ([]PageOutput, error)
	Truncated() bool

	Parent() Page
	Pages() Pages
	PrevInSection() Page
	NextInSection() Page
	Sections(langIndex int) Pages
	RegularPages() Pages
	RegularPagesRecursive() Pages
	Terms(langIndex int, taxonomy string) Pages

	IsTranslated() bool
	Translations() Pages

	Store() *maps.Scratch
	Scratch() *maps.Scratch
}

type OrdinalWeightPage interface {
	Weight() int
	Ordinal() int
	Page() Page
	Owner() Page
}

type PageMeta interface {
	Description() string
	Params() maps.Params
	PageWeight() int
	PageDate() time.Time
	PublishDate() time.Time
	RelatedKeywords(cfg IndexConfig) ([]Keyword, error)

	ShouldList(global bool) bool
	ShouldListAny() bool
	NoLink() bool
}

type PageOutput interface {
	TargetFileBase() string
	TargetFilePath() string
	TargetSubResourceDir() string
	TargetPrefix() string
	TargetFormat() output.Format

	Content() (any, error)
	Summary() goTmpl.HTML
	TableOfContents() goTmpl.HTML
	Result() markdown.Result
	ReadingTime() int
	WordCount() int
}

type PagerManager interface {
	Current() Pager
	SetCurrent(current Pager)

	Paginator() (Pager, error)
	Paginate(groups PageGroups) (Pager, error)
}

type Pagers []Pager

type Pager interface {
	PageNumber() int
	TotalPages() int

	URL() string
	Pages() Pages

	Pagers() Pagers
	First() Pager
	Last() Pager
	HasPrev() bool
	Prev() Pager
	HasNext() bool
	Next() Pager
}

type WalkFunc func(Page) error
type WalkTaxonomyFunc func(taxonomy string, term string, page OrdinalWeightPage) error

// RawContentProvider provides the raw, unprocessed content of the page.
type RawContentProvider interface {
	// RawContent returns the raw, unprocessed content of the page excluding any front matter.
	RawContent() string
	PureContent() string
}

// PageWithoutContent is the Page without any of the content methods.
type PageWithoutContent interface {
	// FileProvider For pages backed by a file.
	FileProvider

	PageMetaProvider
}

// PageMetaProvider provides page metadata, typically provided via front matter.
type PageMetaProvider interface {

	// Kind The Page Kind. One of page, home, section, taxonomy, term.
	Kind() string

	// SectionsEntries Returns a slice of sections (directories if it's a file) to this
	// Page.
	SectionsEntries() []string

	// Layout The configured layout to use to render this page. Typically set in front matter.
	Layout() string

	// Type is a discriminator used to select layouts etc. It is typically set
	// in front matter, but will fall back to the root section.
	Type() string

	Lang() string
	Path() string
}

type ResourceParamsProvider interface {
	// Params set in front matter for this resource.
	Params() maps.Params
}
