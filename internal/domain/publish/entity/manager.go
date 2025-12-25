package entity

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mdfriday/hugoverse/internal/application"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/zip"
	"github.com/spf13/afero"
)

// Manager Publish 业务逻辑管理器
// 负责站点部署、容量管理、自定义域名
type Manager struct {
	fs   afero.Fs
	repo Repository
}

// Repository 数据仓库接口
type Repository interface {
	// License
	GetLicenseByKey(key string) (*contentVO.License, error)

	// PublishSite
	SavePublishSite(site *contentVO.PublishSite) error
	GetPublishSitesByLicense(licenseKey string) ([]contentVO.PublishSite, error)
	GetPublishSiteByName(licenseKey, name string) (*contentVO.PublishSite, error)
	DeletePublishSite(licenseKey, name string) error

	// PublishUsage
	SavePublishUsage(usage *contentVO.PublishUsage) error
	GetPublishUsageByLicense(licenseKey string) (*contentVO.PublishUsage, error)

	// PublishDomain
	SavePublishDomain(domain *contentVO.PublishDomain) error
	GetPublishDomainBySite(licenseKey, siteName string) (*contentVO.PublishDomain, error)
	GetPublishDomainsByLicense(licenseKey string) ([]contentVO.PublishDomain, error)
	GetAllActivePublishDomains() ([]contentVO.PublishDomain, error)
	DeletePublishDomain(licenseKey, domain string) error
}

// NewManager 创建 Publish Manager
func NewManager(repo Repository) *Manager {
	return &Manager{
		fs:   afero.NewOsFs(),
		repo: repo,
	}
}

// NewManagerWithFs 创建带自定义文件系统的 Publish Manager (用于测试)
func NewManagerWithFs(repo Repository, fs afero.Fs) *Manager {
	return &Manager{
		fs:   fs,
		repo: repo,
	}
}

// ========== 站点部署 ==========

// DeploySite 部署站点
func (m *Manager) DeploySite(license *contentVO.License, site *contentVO.PublishSite) error {
	// 检查权限
	features := license.GetFeatures()
	if !features.PublishEnabled {
		return fmt.Errorf("publish not enabled for plan: %s", license.Plan)
	}

	// 检查站点数量限制
	usage, _ := m.GetUsage(license.LicenseKey)
	if usage != nil && !usage.CanAddSite() {
		return fmt.Errorf("site limit reached (%d/%d)", usage.SiteCount, usage.MaxSites)
	}

	userDir := license.ToUserDir()

	// 构建目标目录
	var targetDir string
	if site.SiteType == "article" {
		targetDir = filepath.Join(application.PublishDir(), userDir, "articles", site.Name)
	} else {
		targetDir = filepath.Join(application.PublishDir(), userDir, "sites", site.Name)
	}

	// 确保目录存在
	if err := m.fs.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 解压资源文件
	absAssetPath, err := site.AbsAssetPath(application.UploadDir())
	if err != nil {
		return fmt.Errorf("invalid asset path: %w", err)
	}

	if err := zip.Unzip(absAssetPath, targetDir); err != nil {
		return fmt.Errorf("failed to unzip asset: %w", err)
	}

	// 更新站点信息
	site.License = license.LicenseKey
	site.FolderPath = targetDir
	site.PublicURL = fmt.Sprintf("/%s/%s/%ss/%s",
		application.PublishFolder(), userDir, site.SiteType, site.Name)
	site.Status = "active"
	site.CreatedAt = time.Now().UnixMilli()

	// 计算目录大小
	size, _ := m.getDirSize(targetDir)
	site.Size = size

	if err := m.repo.SavePublishSite(site); err != nil {
		return fmt.Errorf("failed to save site: %w", err)
	}

	// 更新使用量
	m.UpdateUsage(license.LicenseKey)

	return nil
}

// GetSites 获取 License 的所有站点
func (m *Manager) GetSites(licenseKey string) ([]contentVO.PublishSite, error) {
	return m.repo.GetPublishSitesByLicense(licenseKey)
}

// GetSite 获取指定站点
func (m *Manager) GetSite(licenseKey, name string) (*contentVO.PublishSite, error) {
	return m.repo.GetPublishSiteByName(licenseKey, name)
}

// DeleteSite 删除站点
func (m *Manager) DeleteSite(licenseKey, name string) error {
	site, err := m.repo.GetPublishSiteByName(licenseKey, name)
	if err != nil {
		return fmt.Errorf("site not found: %w", err)
	}

	// 删除文件目录
	if site.FolderPath != "" {
		if err := m.fs.RemoveAll(site.FolderPath); err != nil {
			return fmt.Errorf("failed to remove site directory: %w", err)
		}
	}

	// 删除关联的域名
	m.repo.DeletePublishDomain(licenseKey, site.Name)

	// 删除站点记录
	if err := m.repo.DeletePublishSite(licenseKey, name); err != nil {
		return fmt.Errorf("failed to delete site record: %w", err)
	}

	// 更新使用量
	m.UpdateUsage(licenseKey)

	return nil
}

// ========== 容量管理 ==========

// UpdateUsage 更新并记录 Publish 使用量
func (m *Manager) UpdateUsage(licenseKey string) (*contentVO.PublishUsage, error) {
	sites, err := m.repo.GetPublishSitesByLicense(licenseKey)
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for _, site := range sites {
		if site.Status == "active" {
			size, _ := m.getDirSize(site.FolderPath)
			totalSize += size
		}
	}

	// 获取 License 配额
	license, err := m.repo.GetLicenseByKey(licenseKey)
	if err != nil {
		return nil, err
	}
	features := license.GetFeatures()

	usage := &contentVO.PublishUsage{
		License:      licenseKey,
		SiteCount:    len(sites),
		StorageBytes: totalSize,
		MaxSites:     features.MaxSites,
		QuotaBytes:   int64(features.MaxStorageMB) * 1024 * 1024,
		RecordedAt:   time.Now().UnixMilli(),
	}

	if err := m.repo.SavePublishUsage(usage); err != nil {
		return nil, fmt.Errorf("failed to save usage: %w", err)
	}

	return usage, nil
}

// GetUsage 获取当前使用量
func (m *Manager) GetUsage(licenseKey string) (*contentVO.PublishUsage, error) {
	return m.repo.GetPublishUsageByLicense(licenseKey)
}

func (m *Manager) getDirSize(path string) (int64, error) {
	var size int64
	err := afero.Walk(m.fs, path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// ========== 自定义域名 (Caddy 管理 SSL) ==========

// BindCustomDomain 绑定自定义域名
func (m *Manager) BindCustomDomain(license *contentVO.License, siteName, domain string) (*contentVO.PublishDomain, error) {
	// 检查权限
	features := license.GetFeatures()
	if !features.CustomDomain {
		return nil, fmt.Errorf("custom domain not enabled for plan: %s", license.Plan)
	}

	// 获取站点信息
	site, err := m.repo.GetPublishSiteByName(license.LicenseKey, siteName)
	if err != nil {
		return nil, fmt.Errorf("site not found: %w", err)
	}

	now := time.Now().UnixMilli()
	publishDomain := &contentVO.PublishDomain{
		License:     license.LicenseKey,
		PublishSite: siteName,
		Domain:      domain,
		TargetPath:  site.FolderPath,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.repo.SavePublishDomain(publishDomain); err != nil {
		return nil, fmt.Errorf("failed to save domain: %w", err)
	}

	// 更新 Caddy 配置
	if err := m.updateCaddyConfig(); err != nil {
		return nil, fmt.Errorf("failed to update caddy config: %w", err)
	}

	return publishDomain, nil
}

// UnbindCustomDomain 解绑自定义域名
func (m *Manager) UnbindCustomDomain(licenseKey, domain string) error {
	if err := m.repo.DeletePublishDomain(licenseKey, domain); err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	// 更新 Caddy 配置
	return m.updateCaddyConfig()
}

// GetDomains 获取 License 的所有自定义域名
func (m *Manager) GetDomains(licenseKey string) ([]contentVO.PublishDomain, error) {
	return m.repo.GetPublishDomainsByLicense(licenseKey)
}

// updateCaddyConfig 更新 Caddy 配置文件
func (m *Manager) updateCaddyConfig() error {
	// 获取所有活跃的域名配置
	domains, err := m.repo.GetAllActivePublishDomains()
	if err != nil {
		return err
	}

	// 生成 Caddyfile 内容
	var config string
	for _, d := range domains {
		if d.IsActive() {
			config += d.ToCaddyConfig() + "\n\n"
		}
	}

	// 写入 Caddyfile
	caddyfilePath := filepath.Join(application.DataDir(), "Caddyfile")
	if err := afero.WriteFile(m.fs, caddyfilePath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// 注意: 实际环境中需要调用 caddy reload 命令重新加载配置
	// 可以通过 Caddy Admin API 或系统命令实现

	return nil
}

// GenerateCaddyConfig 生成当前所有域名的 Caddy 配置 (用于调试)
func (m *Manager) GenerateCaddyConfig() (string, error) {
	domains, err := m.repo.GetAllActivePublishDomains()
	if err != nil {
		return "", err
	}

	var config string
	for _, d := range domains {
		if d.IsActive() {
			config += d.ToCaddyConfig() + "\n\n"
		}
	}

	return config, nil
}

