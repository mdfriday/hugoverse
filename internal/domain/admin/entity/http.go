package entity

import (
	"fmt"
	"strings"

	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

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

func (h *Http) Domain() string   { return h.Conf.Domain }
func (h *Http) HttpPort() string { return h.Conf.HTTPPort }

// RootDomain extracts the root domain (last two parts) from Domain()
// Examples: abc.example.com -> example.com, sub.abc.example.com -> example.com
func (h *Http) RootDomain() string {
	domain := h.Domain()
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain // already root domain or single part (e.g., localhost)
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

func (h *Http) DevHttpsPort() string { return h.Conf.DevHTTPSPort }
func (h *Http) BindAddress() string  { return h.Conf.BindAddress }
func (h *Http) ServerIP() string     { return h.Conf.ServerIP }

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

func (h *Http) CouchDBDomain() string {
	if h.Env == "dev" {
		return "http://localhost:5984"
	}
	return fmt.Sprintf("https://cdb.%s", h.RootDomain())
}
