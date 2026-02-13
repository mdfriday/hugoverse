package github

import (
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
