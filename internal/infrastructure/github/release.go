package github

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ReleaseDownloader GitHub Release 下载器
type ReleaseDownloader struct {
	client  *http.Client
	timeout time.Duration
}

// NewReleaseDownloader 创建 Release 下载器
func NewReleaseDownloader() *ReleaseDownloader {
	return &ReleaseDownloader{
		client: &http.Client{
			Timeout: 5 * time.Minute, // 下载超时时间
		},
		timeout: 5 * time.Minute,
	}
}

// DownloadZipball 下载 Release 的 zipball
// url: GitHub zipball URL (e.g., https://api.github.com/repos/owner/repo/zipball/v1.0.0)
// destPath: 目标文件路径
func (d *ReleaseDownloader) DownloadZipball(url, destPath, token string) error {
	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub API 需要 User-Agent
	req.Header.Set("User-Agent", "MDFriday-Webhook-Handler")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download zipball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 确保目标目录存在
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建临时文件（下载完成后再重命名，避免下载过程中文件不完整）
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// 写入文件
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tempPath) // 清理临时文件
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 关闭文件
	if err := out.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// 重命名临时文件为目标文件
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// 验证文件大小
	if written == 0 {
		os.Remove(destPath)
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

// DownloadReleaseZip 下载 Release 并保存为指定文件名
func (d *ReleaseDownloader) DownloadReleaseZip(zipballURL, targetDir, filename, token string) error {
	destPath := filepath.Join(targetDir, filename)
	return d.DownloadZipball(zipballURL, destPath, token)
}

// DownloadAsset 下载单个 Release Asset
func (d *ReleaseDownloader) DownloadAsset(url, destPath, token string) error {
	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub 下载不需要特殊认证头（browser_download_url 是公开的）
	req.Header.Set("User-Agent", "MDFriday-Webhook-Handler")

	// 如果提供了 token，添加认证（用于私有仓库）
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// 发送请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 确保目标目录存在
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建临时文件
	tempPath := destPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// 写入文件
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 关闭文件
	if err := out.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// 重命名临时文件为目标文件
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// 验证文件大小
	if written == 0 {
		os.Remove(destPath)
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}

// DownloadAssetsAsZip 下载所有 Release Assets 并打包成 ZIP
func (d *ReleaseDownloader) DownloadAssetsAsZip(assets []ReleaseAsset, targetDir, zipFilename, token string) error {
	if len(assets) == 0 {
		return fmt.Errorf("no assets to download")
	}

	// 创建临时目录存放下载的文件
	tempDir := filepath.Join(targetDir, ".temp_assets")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 下载所有 assets
	for _, asset := range assets {
		assetPath := filepath.Join(tempDir, asset.Name)
		if err := d.DownloadAsset(asset.BrowserDownloadURL, assetPath, token); err != nil {
			return fmt.Errorf("failed to download asset %s: %w", asset.Name, err)
		}
	}

	// 打包成 ZIP
	zipPath := filepath.Join(targetDir, zipFilename)
	if err := d.createZipFromDir(tempDir, zipPath); err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}

	return nil
}

// createZipFromDir 将目录打包成 ZIP 文件
func (d *ReleaseDownloader) createZipFromDir(sourceDir, zipPath string) error {
	// 创建临时 ZIP 文件
	tempZipPath := zipPath + ".tmp"
	zipFile, err := os.Create(tempZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历源目录
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// 在 ZIP 中创建文件
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// 读取源文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 写入 ZIP
		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		os.Remove(tempZipPath)
		return err
	}

	// 关闭 ZIP writer
	if err := zipWriter.Close(); err != nil {
		os.Remove(tempZipPath)
		return err
	}

	// 关闭文件
	if err := zipFile.Close(); err != nil {
		os.Remove(tempZipPath)
		return err
	}

	// 重命名临时文件为目标文件
	if err := os.Rename(tempZipPath, zipPath); err != nil {
		os.Remove(tempZipPath)
		return err
	}

	return nil
}
