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

// CouchDBDomain 返回 CouchDB 对外访问地址（给客户端使用）
// 对内连接地址使用 CouchDB entity 的 CouchDBURL() 方法
func (h *Http) CouchDBDomain() string {
	subdomain := h.Conf.CouchDBSubDomain
	if subdomain == "" {
		subdomain = "cdb" // 默认值
	}

	domain := h.Domain()
	isLocalhost := (domain == "localhost" || domain == "127.0.0.1")

	if h.Env == "dev" || isLocalhost {
		// 开发环境或 localhost：使用 HTTP + Caddy 对外端口
		externalPort := h.Conf.ExternalHTTPPort
		if externalPort == "" || externalPort == "80" {
			return fmt.Sprintf("http://%s.%s", subdomain, domain)
		}
		return fmt.Sprintf("http://%s.%s:%s", subdomain, domain, externalPort)
	}

	// 生产环境：使用 HTTPS + 根域名
	return fmt.Sprintf("https://%s.%s", subdomain, h.RootDomain())
}

// HugoverseDomain 返回 Hugoverse 对外访问地址（给客户端使用）
// 对内连接地址使用 hugoverse:1314
func (h *Http) HugoverseDomain() string {
	subdomain := h.Conf.HugoverseSubDomain
	if subdomain == "" {
		subdomain = "app" // 默认值
	}

	domain := h.Domain()
	isLocalhost := (domain == "localhost" || domain == "127.0.0.1")

	if h.Env == "dev" || isLocalhost {
		// 开发环境或 localhost：使用 HTTP + Caddy 对外端口
		externalPort := h.Conf.ExternalHTTPPort
		if externalPort == "" || externalPort == "80" {
			return fmt.Sprintf("http://%s.%s", subdomain, domain)
		}
		return fmt.Sprintf("http://%s.%s:%s", subdomain, domain, externalPort)
	}

	// 生产环境：使用 HTTPS + 根域名
	return fmt.Sprintf("https://%s.%s", subdomain, h.RootDomain())
}
