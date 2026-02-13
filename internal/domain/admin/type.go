package admin

import (
	"net/url"
)

type Admin interface {
	Name() string

	Editor
	UserService
	Persistence
	Cache
	Controller
	Upload
	Http
	Client
	GitHub
}

type Traceable interface {
	FilePath() string
}

type Editor interface {
	ConfigEditor() ([]byte, error)
}

type User interface {
	Name() string
	ID() uint64
}

type UserService interface {
	ValidateUser(email, password string) error
	NewUser(email, password string) (User, error)
	IsUserExists(email string) bool
}

type Upload interface {
	UploadCreator() func() interface{}
}

type Persistence interface {
	PutConfig(key string, value any) error
	SetConfig(data url.Values) error
	NewUpload(data url.Values) error
	GetUpload(id string) ([]byte, error)
	DeleteUpload(id string) error
	AllUploads() ([][]byte, error)
}

type Http interface {
	Domain() string
	HttpPort() string
	BindAddress() string
	DevHttpsPort() string
}

type Cache interface {
	InvalidateCache() error
	ETage() string
	NewETage() string
	CacheMaxAge() int64
}

type Controller interface {
	CacheDisabled() bool
	CorsDisabled() bool
	GzipDisabled() bool
}

type Client interface {
	ClientSecret() string
}

type GitHub interface {
	HookSecret() string
	GithubToken() string
	TargetRepository() string
}
