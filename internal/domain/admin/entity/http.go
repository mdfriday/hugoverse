package entity

import "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"

type Http struct {
	Env  string
	Conf *valueobject.Config
}

func (h *Http) Host() string {
	if h.Env == "dev" {
		return "site.test"
	}
	return h.Conf.Domain
}

func (h *Http) Domain() string       { return h.Conf.Domain }
func (h *Http) HttpPort() string     { return h.Conf.HTTPPort }
func (h *Http) DevHttpsPort() string { return h.Conf.DevHTTPSPort }
func (h *Http) BindAddress() string  { return h.Conf.BindAddress }

func (h *Http) DevPort() string {
	return "8080"
}

func (h *Http) SubBaseURL(sub string) string {
	if h.Env == "dev" {
		return "http://" + sub + "." + h.Host() + ":" + h.DevPort()
	}
	return "https://" + sub + "." + h.Host()
}

func (h *Http) BaseURL() string {
	if h.Env == "dev" {
		return "http://" + h.Host() + ":" + h.DevPort()
	}
	return "https://" + h.Host()
}
