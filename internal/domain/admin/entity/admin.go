package entity

import (
	"encoding/json"
	"net/url"

	"github.com/mdfriday/hugoverse/internal/domain/admin/repository"
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

type Admin struct {
	Repo repository.Repository
	Conf *valueobject.Config

	*Administrator
	*Upload
	*Http
	*Cache
	*Controller
	*Client
	*Netlify
	*CouchDB
	*Caddy

	Log loggers.Logger
}

func (a *Admin) ConfigEditor() ([]byte, error) {
	// TODO, remove it
	return a.Conf.MarshalEditor()
}

func (a *Admin) SetConfig(data url.Values) error {
	c, err := a.Conf.Convert(data)
	if err != nil {
		return err
	}

	m, err := c.Marshal()
	if err != nil {
		return err
	}

	if err := a.Repo.PutConfig(m); err != nil {
		return err
	}
	if err := a.LoadConfig(); err != nil {
		return err
	}
	// check for "invalidate" value to reset the Etag
	if a.Conf.IsCacheInvalidate() {
		a.RefreshETage()
	}

	return nil
}

func (a *Admin) LoadConfig() error {
	var conf *valueobject.Config

	data, err := a.Repo.LoadConfig()
	if err != nil {
		return err
	}

	if data == nil {
		conf = &valueobject.Config{}
	} else {
		err = json.Unmarshal(data, &conf)
		if err != nil {
			return err
		}
	}

	a.Conf = conf
	return nil
}

func (a *Admin) PutConfig(key string, value any) error {
	c, err := a.Conf.Update(key, value)
	if err != nil {
		return err
	}

	j, err := c.Marshal()
	if err != nil {
		return err
	}

	err = a.Repo.PutConfig(j)
	if err != nil {
		return err
	}

	a.Conf = c

	// Update Config reference in embedded structs
	if a.Http != nil {
		a.Http.Conf = a.Conf
	}
	if a.Cache != nil {
		a.Cache.Conf = a.Conf
	}
	if a.Controller != nil {
		a.Controller.Conf = a.Conf
	}
	if a.Client != nil {
		a.Client.Conf = a.Conf
	}
	if a.Netlify != nil {
		a.Netlify.Conf = a.Conf
	}
	if a.CouchDB != nil {
		a.CouchDB.Conf = a.Conf
	}

	return nil
}

func (a *Admin) RefreshETage() {
	a.Conf.Etag = valueobject.NewEtag()
	a.Conf.CacheInvalidate = []string{}
}

func (a *Admin) InvalidateCache() error {
	a.RefreshETage()
	err := a.PutConfig("etag", a.Conf.Etag)

	if err != nil {
		return err
	}

	return nil
}

func (a *Admin) Name() string { return a.Conf.Name }
