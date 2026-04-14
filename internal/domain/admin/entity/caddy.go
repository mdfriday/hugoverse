package entity

import (
	"fmt"
	"os"
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

type Caddy struct {
	Env  string
	Conf *valueobject.Config
}

func (c *Caddy) CaddyHost() string { return c.Conf.CaddyHost }
func (c *Caddy) CaddyPort() string { return c.Conf.CaddyPort }

// CaddyAdminAPI 返回 Caddy Admin API 地址（内部连接使用）
func (c *Caddy) CaddyAdminAPI() string {
	host := c.CaddyHost()
	port := c.CaddyPort()
	
	// 如果配置为空，从环境变量读取（Docker 环境或首次启动）
	if host == "" {
		host = os.Getenv("CADDY_HOST")
		if host == "" {
			host = "127.0.0.1" // 最后的默认值
		}
	}
	if port == "" {
		port = os.Getenv("CADDY_PORT")
		if port == "" {
			port = "2019"
		}
	}
	
	return fmt.Sprintf("http://%s:%s", host, port)
}

// CaddyURL 返回 Caddy 对外访问地址（已废弃，保留向后兼容）
// 推荐使用 CaddyAdminAPI() 获取 Admin API 地址
func (c *Caddy) CaddyURL() string {
	return c.CaddyAdminAPI()
}

