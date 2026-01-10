package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/entity"
	"github.com/mdfriday/hugoverse/internal/domain/content/repository"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/spf13/afero"
)

func NewContent(repo repository.Repository, dir content.DirService) *entity.Content {
	log := loggers.NewDefault()
	log.Debugln("user data dir: ", repo.UserDataDir())

	c := &entity.Content{
		UserTypes:  make(map[string]content.Creator),
		AdminTypes: make(map[string]content.Creator),
		Repo:       repo,

		Hugo: &entity.Hugo{
			Fs:         afero.NewOsFs(),
			DirService: dir,

			Log: log,
		},

		Log: log,
	}

	prepareUserTypes(c)
	prepareAdminTypes(c)

	c.Search = &entity.Search{
		TypeService: c,
		Repo:        repo,
		Log:         log,

		IndicesMap: make(map[string]*entity.CacheIndex),
	}

	return c
}

func prepareUserTypes(c *entity.Content) {
	//c.UserTypes["Author"] = func() interface{} { return new(valueobject.Author) }
	//c.UserTypes["Language"] = func() interface{} { return new(valueobject.Language) }
	//c.UserTypes["Post"] = func() interface{} { return new(valueobject.Post) }
	//c.UserTypes["Resource"] = func() interface{} { return new(valueobject.Resource) }
	//c.UserTypes["Site"] = func() interface{} { return new(valueobject.Site) }
	//c.UserTypes["SiteLanguage"] = func() interface{} { return new(valueobject.SiteLanguage) }
	//c.UserTypes["SitePost"] = func() interface{} { return new(valueobject.SitePost) }
	//c.UserTypes["SiteResource"] = func() interface{} { return new(valueobject.SiteResource) }
	//c.UserTypes["Deployment"] = func() interface{} { return new(valueobject.Deployment) }
	c.UserTypes["CTA"] = func() interface{} { return new(valueobject.CTA) }
	//c.AdminTypes["Domain"] = func() interface{} { return new(valueobject.Domain) }
}

func prepareAdminTypes(c *entity.Content) {
	c.AdminTypes["Preview"] = func() interface{} { return new(valueobject.Preview) }
	c.AdminTypes["MDFPreview"] = func() interface{} { return new(valueobject.MDFPreview) }
	c.AdminTypes["Image"] = func() interface{} { return new(valueobject.Image) }
	//c.AdminTypes["ShortCode"] = func() interface{} { return new(valueobject.ShortCode) }
	//c.AdminTypes["Theme"] = func() interface{} { return new(valueobject.Theme) }
	c.AdminTypes["Counter"] = func() interface{} { return new(valueobject.Counter) }

	// License 管理
	c.AdminTypes["License"] = func() interface{} { return new(valueobject.License) }
	c.AdminTypes["LicenseDevice"] = func() interface{} { return new(valueobject.LicenseDevice) }
	c.AdminTypes["LicenseIP"] = func() interface{} { return new(valueobject.LicenseIP) }
	c.AdminTypes["LicenseUsage"] = func() interface{} { return new(valueobject.LicenseUsage) }

	// Sync 相关
	c.AdminTypes["SyncAccount"] = func() interface{} { return new(valueobject.SyncAccount) }

	// Publish 相关
	c.AdminTypes["PublishSite"] = func() interface{} { return new(valueobject.PublishSite) }
	c.AdminTypes["SubDomain"] = func() interface{} { return new(valueobject.SubDomain) }
	c.AdminTypes["PublishDomain"] = func() interface{} { return new(valueobject.PublishDomain) }
}

func NewContentWithServices(repo repository.Repository, services content.Services, dirService content.DirService) *entity.Content {
	c := NewContent(repo, dirService)
	c.Hugo.Services = services

	return c
}

func NewItem() (*valueobject.Item, error) {
	return valueobject.NewItem()
}
