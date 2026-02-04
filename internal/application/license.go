package application

import (
	"encoding/json"
	"time"

	contentEntity "github.com/mdfriday/hugoverse/internal/domain/content/entity"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
)

func LicenseResourceRecycle(cs *contentEntity.Content, log loggers.Logger) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop() // 确保在程序退出时停止定时器

	log.Println("The license resource cleanup task has been initiated and will run once every 12 hours...")

	for {
		select {
		case t := <-ticker.C:
			log.Println("Recycle task triggered:", t)
			recycleLicenseResources(cs, log) // 执行回收逻辑
		}
	}
}

// recyclePreviewSites 执行预览站点的回收逻辑
func recycleLicenseResources(cs *contentEntity.Content, log loggers.Logger) {
	ns := "License"
	all := cs.Repo.AllContent(ns)
	p, ok := cs.AllAdminTypes()[ns]
	if !ok {
		log.Errorf("Type %s not supported", ns)
		return
	}

	ct := timestamp.Now()

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
					t, err := timestamp.ConvertInt64ToTime(license.ExpiryDate)
					if err != nil {
						log.Errorln("Error converting time when recycling ", ns, err)
					}
					log.Errorf("License: %s, ExpiryDate: %s, Check time: %s", license.LicenseKey, t.String(), ct)
				}
			}
		} else {
			log.Errorf("Type assertion failed for license %d when recycling %s", i, ns)
		}
	}
}
