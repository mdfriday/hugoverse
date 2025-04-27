package images

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/webp"
)

func GetImageDimensions(absPath string) (int, int, error) {
	// 打开图片文件
	file, err := os.Open(absPath)
	if err != nil {
		return 0, 0, fmt.Errorf("can not open image: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(absPath))

	var config image.Config
	var format string
	var decodeErr error

	switch ext {
	case ".webp":
		config, decodeErr = webp.DecodeConfig(file)
		format = "webp"
	case ".jpg", ".jpeg", ".png", ".gif":
		config, format, decodeErr = image.DecodeConfig(file)
	default:
		return 0, 0, fmt.Errorf("unsupported image format: %s", ext)
	}

	if decodeErr != nil {
		return 0, 0, fmt.Errorf("can not decode image (%s): %w", format, decodeErr)
	}

	return config.Width, config.Height, nil
}
