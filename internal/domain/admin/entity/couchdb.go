package entity

import (
	"os"
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

type CouchDB struct {
	Conf *valueobject.Config
}

// CouchDBURL 返回 CouchDB 内部连接地址
func (cdb *CouchDB) CouchDBURL() string {
	url := cdb.Conf.URL
	if url == "" {
		// 从环境变量读取（首次启动时配置可能为空）
		url = os.Getenv("COUCHDB_URL")
		if url == "" {
			url = "http://localhost:5984" // 最后的默认值
		}
	}
	return url
}

// CouchDBAdminName 返回 CouchDB 管理员用户名
func (cdb *CouchDB) CouchDBAdminName() string {
	user := cdb.Conf.AdminUser
	if user == "" {
		user = os.Getenv("COUCHDB_USER")
		if user == "" {
			user = "admin"
		}
	}
	return user
}

// CouchDBAdminPassword 返回 CouchDB 管理员密码
func (cdb *CouchDB) CouchDBAdminPassword() string {
	pass := cdb.Conf.AdminPass
	if pass == "" {
		pass = os.Getenv("COUCHDB_PASSWORD")
	}
	return pass
}

// CouchDBPrefix 返回用户数据库前缀
func (cdb *CouchDB) CouchDBPrefix() string {
	prefix := cdb.Conf.DBPrefix
	if prefix == "" {
		prefix = os.Getenv("COUCHDB_DB_PREFIX")
		if prefix == "" {
			prefix = "userdb-"
		}
	}
	return prefix
}
