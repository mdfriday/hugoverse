package entity

import (
	"fmt"
	"testing"
	"time"

	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/spf13/afero"
)

// MockPublishRepository 模拟数据仓库
type MockPublishRepository struct {
	licenses       map[string]*contentVO.License
	sites          map[string]*contentVO.PublishSite
	usages         map[string]*contentVO.PublishUsage
	domains        map[string]*contentVO.PublishDomain
	activeDomains  []contentVO.PublishDomain
}

func NewMockPublishRepository() *MockPublishRepository {
	return &MockPublishRepository{
		licenses:      make(map[string]*contentVO.License),
		sites:         make(map[string]*contentVO.PublishSite),
		usages:        make(map[string]*contentVO.PublishUsage),
		domains:       make(map[string]*contentVO.PublishDomain),
		activeDomains: make([]contentVO.PublishDomain, 0),
	}
}

func (r *MockPublishRepository) GetLicenseByKey(key string) (*contentVO.License, error) {
	if l, ok := r.licenses[key]; ok {
		return l, nil
	}
	return nil, fmt.Errorf("license not found")
}

func (r *MockPublishRepository) SavePublishSite(site *contentVO.PublishSite) error {
	key := site.License + ":" + site.Name
	r.sites[key] = site
	return nil
}

func (r *MockPublishRepository) GetPublishSitesByLicense(licenseKey string) ([]contentVO.PublishSite, error) {
	var sites []contentVO.PublishSite
	for _, s := range r.sites {
		if s.License == licenseKey {
			sites = append(sites, *s)
		}
	}
	return sites, nil
}

func (r *MockPublishRepository) GetPublishSiteByName(licenseKey, name string) (*contentVO.PublishSite, error) {
	key := licenseKey + ":" + name
	if s, ok := r.sites[key]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("site not found")
}

func (r *MockPublishRepository) DeletePublishSite(licenseKey, name string) error {
	key := licenseKey + ":" + name
	delete(r.sites, key)
	return nil
}

func (r *MockPublishRepository) SavePublishUsage(usage *contentVO.PublishUsage) error {
	r.usages[usage.License] = usage
	return nil
}

func (r *MockPublishRepository) GetPublishUsageByLicense(licenseKey string) (*contentVO.PublishUsage, error) {
	if u, ok := r.usages[licenseKey]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("usage not found")
}

func (r *MockPublishRepository) SavePublishDomain(domain *contentVO.PublishDomain) error {
	key := domain.License + ":" + domain.Domain
	r.domains[key] = domain
	if domain.Status == "active" {
		r.activeDomains = append(r.activeDomains, *domain)
	}
	return nil
}

func (r *MockPublishRepository) GetPublishDomainBySite(licenseKey, siteName string) (*contentVO.PublishDomain, error) {
	for _, d := range r.domains {
		if d.License == licenseKey && d.PublishSite == siteName {
			return d, nil
		}
	}
	return nil, fmt.Errorf("domain not found")
}

func (r *MockPublishRepository) GetPublishDomainsByLicense(licenseKey string) ([]contentVO.PublishDomain, error) {
	var domains []contentVO.PublishDomain
	for _, d := range r.domains {
		if d.License == licenseKey {
			domains = append(domains, *d)
		}
	}
	return domains, nil
}

func (r *MockPublishRepository) GetAllActivePublishDomains() ([]contentVO.PublishDomain, error) {
	return r.activeDomains, nil
}

func (r *MockPublishRepository) DeletePublishDomain(licenseKey, domain string) error {
	key := licenseKey + ":" + domain
	delete(r.domains, key)
	// 更新 activeDomains
	var newActive []contentVO.PublishDomain
	for _, d := range r.activeDomains {
		if !(d.License == licenseKey && d.Domain == domain) {
			newActive = append(newActive, d)
		}
	}
	r.activeDomains = newActive
	return nil
}

// ========== 测试 ==========

func TestPublishManagerUpdateUsage(t *testing.T) {
	repo := NewMockPublishRepository()
	fs := afero.NewMemMapFs()
	manager := NewManagerWithFs(repo, fs)

	license := &contentVO.License{
		LicenseKey: "MDF-PUB-TEST-1234",
		Plan:       contentVO.PlanStarter,
		Activated:  true,
		ExpiryDate: time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	// 添加一些站点
	repo.sites["MDF-PUB-TEST-1234:my-blog"] = &contentVO.PublishSite{
		License:    license.LicenseKey,
		Name:       "my-blog",
		SiteType:   "site",
		FolderPath: "/data/publish/user123/sites/my-blog",
		Status:     "active",
	}

	// 创建测试目录和文件
	fs.MkdirAll("/data/publish/user123/sites/my-blog", 0755)
	afero.WriteFile(fs, "/data/publish/user123/sites/my-blog/index.html", []byte("<html></html>"), 0644)

	// 更新使用量
	usage, err := manager.UpdateUsage(license.LicenseKey)
	if err != nil {
		t.Errorf("UpdateUsage() error = %v", err)
	}

	if usage == nil {
		t.Fatal("Expected usage, got nil")
	}

	if usage.SiteCount != 1 {
		t.Errorf("SiteCount = %d, want 1", usage.SiteCount)
	}
}

func TestPublishManagerBindCustomDomain(t *testing.T) {
	repo := NewMockPublishRepository()
	fs := afero.NewMemMapFs()
	manager := NewManagerWithFs(repo, fs)

	license := &contentVO.License{
		LicenseKey: "MDF-DOMAIN-TEST",
		Plan:       contentVO.PlanCreator, // Creator 支持自定义域名
		Activated:  true,
		ExpiryDate: time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	// 添加站点
	repo.sites["MDF-DOMAIN-TEST:my-blog"] = &contentVO.PublishSite{
		License:    license.LicenseKey,
		Name:       "my-blog",
		SiteType:   "site",
		FolderPath: "/data/publish/user123/sites/my-blog",
		Status:     "active",
	}

	// 绑定域名
	domain, err := manager.BindCustomDomain(license, "my-blog", "blog.example.com")
	if err != nil {
		t.Errorf("BindCustomDomain() error = %v", err)
	}

	if domain == nil {
		t.Fatal("Expected domain, got nil")
	}

	if domain.Domain != "blog.example.com" {
		t.Errorf("Domain = %v, want blog.example.com", domain.Domain)
	}

	if domain.Status != "active" {
		t.Errorf("Status = %v, want active", domain.Status)
	}
}

func TestPublishManagerBindCustomDomainFreePlan(t *testing.T) {
	repo := NewMockPublishRepository()
	fs := afero.NewMemMapFs()
	manager := NewManagerWithFs(repo, fs)

	license := &contentVO.License{
		LicenseKey: "MDF-FREE-DOMAIN",
		Plan:       contentVO.PlanFree, // Free 不支持自定义域名
		Activated:  true,
		ExpiryDate: time.Now().Add(7 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	repo.sites["MDF-FREE-DOMAIN:my-blog"] = &contentVO.PublishSite{
		License: license.LicenseKey,
		Name:    "my-blog",
	}

	// 尝试绑定域名应该失败
	_, err := manager.BindCustomDomain(license, "my-blog", "blog.example.com")
	if err == nil {
		t.Error("Expected error for Free plan, got nil")
	}
}

func TestPublishManagerGenerateCaddyConfig(t *testing.T) {
	repo := NewMockPublishRepository()
	fs := afero.NewMemMapFs()
	manager := NewManagerWithFs(repo, fs)

	// 添加活跃的域名
	repo.activeDomains = []contentVO.PublishDomain{
		{
			License:    "MDF-TEST-1",
			Domain:     "blog.example.com",
			TargetPath: "/data/publish/user1/sites/blog",
			Status:     "active",
		},
		{
			License:    "MDF-TEST-2",
			Domain:     "shop.example.com",
			TargetPath: "/data/publish/user2/sites/shop",
			Status:     "active",
		},
	}

	config, err := manager.GenerateCaddyConfig()
	if err != nil {
		t.Errorf("GenerateCaddyConfig() error = %v", err)
	}

	// 验证配置包含域名
	if config == "" {
		t.Error("Expected non-empty config")
	}

	// 验证配置格式
	expectedContains := []string{
		"blog.example.com",
		"shop.example.com",
		"file_server",
	}

	for _, expected := range expectedContains {
		if !containsString(config, expected) {
			t.Errorf("Config should contain %q", expected)
		}
	}
}

func TestPublishManagerGetDomains(t *testing.T) {
	repo := NewMockPublishRepository()
	fs := afero.NewMemMapFs()
	manager := NewManagerWithFs(repo, fs)

	licenseKey := "MDF-DOMAINS-TEST"

	// 添加多个域名
	repo.domains["MDF-DOMAINS-TEST:blog.example.com"] = &contentVO.PublishDomain{
		License: licenseKey,
		Domain:  "blog.example.com",
		Status:  "active",
	}
	repo.domains["MDF-DOMAINS-TEST:shop.example.com"] = &contentVO.PublishDomain{
		License: licenseKey,
		Domain:  "shop.example.com",
		Status:  "active",
	}

	domains, err := manager.GetDomains(licenseKey)
	if err != nil {
		t.Errorf("GetDomains() error = %v", err)
	}

	if len(domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(domains))
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

