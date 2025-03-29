package images

import (
	"fmt"
	"image"
	"os"
)

func GetImageDimensions(absPath string) (int, int, error) {
	// 打开图片文件
	file, err := os.Open(absPath)
	if err != nil {
		return 0, 0, fmt.Errorf("can not open image: %w", err)
	}
	defer file.Close()

	// 解析图片信息
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("can not decode image: %w", err)
	}

	return config.Width, config.Height, nil
}
