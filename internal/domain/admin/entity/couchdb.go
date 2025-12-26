package entity

import (
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

type CouchDB struct {
	Conf *valueobject.Config
}

func (cdb *CouchDB) CouchDBURL() string           { return cdb.Conf.URL }
func (cdb *CouchDB) CouchDBAdminName() string     { return cdb.Conf.AdminUser }
func (cdb *CouchDB) CouchDBAdminPassword() string { return cdb.Conf.AdminPass }
func (cdb *CouchDB) CouchDBPrefix() string        { return cdb.Conf.DBPrefix }
