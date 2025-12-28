package entity

import "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"

type Http struct {
	Env  string
	Conf *valueobject.Config
}

func (a *Admin) Host() string {
	if a.Env == "dev" {
		return "site.test"
	}
	return a.Conf.Domain
}

func (a *Admin) Domain() string       { return a.Conf.Domain }
func (a *Admin) HttpPort() string     { return a.Conf.HTTPPort }
func (a *Admin) DevHttpsPort() string { return a.Conf.DevHTTPSPort }
func (a *Admin) BindAddress() string  { return a.Conf.BindAddress }

func (a *Admin) DevPort() string {
	return "8080"
}

func (a *Admin) SubBaseURL(sub string) string {
	if a.Env == "dev" {
		return "http://" + sub + "." + a.Host() + ":" + a.DevPort()
	}
	return "https://" + sub + "." + a.Host()
}

func (a *Admin) BaseURL() string {
	if a.Env == "dev" {
		return "http://" + a.Host() + ":" + a.DevPort()
	}
	return "https://" + a.Host()
}
