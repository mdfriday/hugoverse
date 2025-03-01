package fs

import (
	"os"
	"path/filepath"
)

// GetTotalSize calculates the total size of a directory
func GetTotalSize(path string) int64 {
	var totalSize int64
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}
