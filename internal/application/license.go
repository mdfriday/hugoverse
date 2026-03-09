package application

import (
	"encoding/json"
	adminEntity "github.com/mdfriday/hugoverse/internal/domain/admin/entity"
	"github.com/mdfriday/hugoverse/internal/infrastructure/couchdb"
	"path/filepath"
	"time"

	contentEntity "github.com/mdfriday/hugoverse/internal/domain/content/entity"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
)

func LicenseResourceRecycle(cs *contentEntity.Content, adminApp *adminEntity.Admin, log loggers.Logger) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop() // 确保在程序退出时停止定时器

	log.Println("The license resource cleanup task has been initiated and will run once every 12 hours...")

	for {
		select {
		case t := <-ticker.C:
			log.Println("Recycle task triggered:", t)
			recycleLicenseResources(cs, adminApp, log) // 执行回收逻辑
		}
	}
}

// recyclePreviewSites 执行预览站点的回收逻辑
func recycleLicenseResources(cs *contentEntity.Content, adminApp *adminEntity.Admin, log loggers.Logger) {
	ns := "License"
	all := cs.Repo.AllContent(ns)
	p, ok := cs.AllAdminTypes()[ns]
	if !ok {
		log.Errorf("Type %s not supported", ns)
		return
	}

	ct := timestamp.Now()

	var expiredLicenses []*valueobject.License
	for i, v := range all {
		post := p()
		err := json.Unmarshal(v, post)
		if err != nil {
			log.Errorf("Error unmarshalling license %d when recycling: %v", i, err)
			continue
		}

		if license, ok := post.(*valueobject.License); ok {
			if license.Activated {
				if license.IsExpired() {
					if license.ActivatedAt == 0 {
						continue
					}
					t, err := timestamp.ConvertInt64ToTime(license.ExpiryDate)
					if err != nil {
						log.Errorln("Error converting time when recycling ", ns, err)
					}
					log.Errorf("License: %s, ExpiryDate: %s, Check time: %s", license.LicenseKey, t.String(), ct)
					expiredLicenses = append(expiredLicenses, license)
				}
			}
		} else {
			log.Errorf("Type assertion failed for license %d when recycling %s", i, ns)
		}
	}

	if len(expiredLicenses) > 0 {
		couchClient := couchdb.NewClient(&couchdb.Config{
			URL:       adminApp.CouchDBURL(),
			AdminUser: adminApp.CouchDBAdminName(),
			AdminPass: adminApp.CouchDBAdminPassword(),
			DBPrefix:  adminApp.CouchDBPrefix(),
		})

		for _, license := range expiredLicenses {
			syncAccount, err := cs.GetSyncAccountByLicense(license.LicenseKey)
			if err == nil && syncAccount != nil {
				dbName := syncAccount.DBName
				if err := couchClient.DeleteDatabase(dbName); err != nil {
					log.Errorf("Failed to delete CouchDB database %s: %v", dbName, err)
				}
				log.Printf("  CouchDB database %s associated with license %s has been deleted.", dbName, license.LicenseKey)
			} else {
				log.Errorf("Failed to get sync account for license %s: %v", license.LicenseKey, err)
			}

			if err = DeleteDir(filepath.Join(PreviewDir(), license.ToUserShortDir())); err != nil {
				log.Errorf("Failed to delete preview directory for license %s: %v", license.LicenseKey, err)
			} else {
				log.Printf("  Preview directory for license %s has been deleted.", license.LicenseKey)
			}

			license.ActivatedAt = 0
			if err := cs.UpdateLicense(license); err != nil {
				log.Errorf("Failed to update license: %v", err)
			} else {
				log.Printf("  License %s has been reset to pending state.", license.LicenseKey)
			}

			log.Printf("License %s has expired and its associated resources have been recycled.", license.LicenseKey)
		}
	}
}
