package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Unzip(src string, dest string) error {
	// 打开 zip 文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("无法打开 zip 文件: %w", err)
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			fmt.Printf("关闭 zip 文件时出错: %v\n", err)
		}
	}(r)

	// 遍历 zip 中的每个文件
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 防止 Zip Slip
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法文件路径：%s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			err := rc.Close()
			if err != nil {
				return err
			}
			return err
		}

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}

		// 手动关闭资源
		err = outFile.Close()
		if err != nil {
			return err
		}
		err = rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
