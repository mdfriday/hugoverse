package application

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// FridayResourceRecycle 定期清理 Friday 免费预览目录
// 每小时检查一次，删除超过 24 小时的文件夹
func FridayResourceRecycle(log loggers.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("The Friday resource cleanup task has been initiated and will run once every hour...")

	for {
		select {
		case t := <-ticker.C:
			log.Println("Friday recycle task triggered:", t)
			recycleFridayResources(log)
		}
	}
}

// recycleFridayResources 执行 Friday 目录的清理逻辑
func recycleFridayResources(log loggers.Logger) {
	fridayDir := FridayDir()

	// 读取 Friday 目录下的所有文件夹
	entries, err := os.ReadDir(fridayDir)
	if err != nil {
		log.Errorf("Failed to read Friday directory %s: %v", fridayDir, err)
		return
	}

	now := time.Now()
	deletedCount := 0
	totalCount := 0

	for _, entry := range entries {
		// 只处理目录
		if !entry.IsDir() {
			continue
		}

		totalCount++
		dirPath := filepath.Join(fridayDir, entry.Name())

		// 获取目录信息
		info, err := entry.Info()
		if err != nil {
			log.Errorf("Failed to get info for directory %s: %v", dirPath, err)
			continue
		}

		// 检查创建/修改时间
		modTime := info.ModTime()
		age := now.Sub(modTime)

		// 如果超过 24 小时，删除目录
		if age > 24*time.Hour {
			if err := DeleteDir(dirPath); err != nil {
				log.Errorf("Failed to delete expired Friday directory %s (age: %v): %v", dirPath, age, err)
			} else {
				deletedCount++
				log.Printf("Deleted expired Friday directory: %s (age: %v)", entry.Name(), age)
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("Friday cleanup completed: deleted %d of %d directories", deletedCount, totalCount)
	} else {
		log.Printf("Friday cleanup completed: no expired directories found (total: %d)", totalCount)
	}
}
