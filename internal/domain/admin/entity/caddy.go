package entity

import (
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

type Caddy struct {
	Env  string
	Conf *valueobject.Config
}

func (c *Caddy) CaddyHost() string { return c.Conf.CaddyHost }
func (c *Caddy) CaddyPort() string { return c.Conf.CaddyPort }

func (c *Caddy) CaddyURL() string {
	if c.Env == "dev" {
		return "http://" + c.CaddyHost() + ":" + c.CaddyPort()
	}
	return "https://" + c.CaddyHost()
}
