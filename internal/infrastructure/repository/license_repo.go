package repository

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	publishEntity "github.com/mdfriday/hugoverse/internal/domain/publish/entity"
	syncEntity "github.com/mdfriday/hugoverse/internal/domain/sync/entity"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/database"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

const (
	NsLicense       = "License"
	NsLicenseDevice = "LicenseDevice"
	NsLicenseIP     = "LicenseIP"
	NsSyncAccount   = "SyncAccount"
	NsSyncUsage     = "SyncUsage"
	NsPublishSite   = "PublishSite"
	NsPublishUsage  = "PublishUsage"
	NsPublishDomain = "PublishDomain"
)

// LicenseRepository License 相关的数据仓库实现
type LicenseRepository struct {
	db *database.Database
}

// NewLicenseRepository 创建 LicenseRepository
func NewLicenseRepository(db *database.Database) *LicenseRepository {
	return &LicenseRepository{db: db}
}

// ========== License 操作 ==========

// GetLicenseByKey 通过 LicenseKey 查找 License
func (r *LicenseRepository) GetLicenseByKey(licenseKey string) (*valueobject.License, error) {
	hashKey := hash.MD5(licenseKey)

	idBytes, err := r.db.GetIdByHash(NsLicense, hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("license not found: %s", licenseKey)
	}

	data, err := r.db.GetContent(NsLicense, string(idBytes))
	if err != nil {
		return nil, err
	}

	var license valueobject.License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, err
	}

	return &license, nil
}

// UpdateLicense 更新 License
func (r *LicenseRepository) UpdateLicense(license *valueobject.License) error {
	data, err := json.Marshal(license)
	if err != nil {
		return err
	}
	return r.db.PutContent(license, data)
}

// CreateLicense 创建新的 License
func (r *LicenseRepository) CreateLicense(license *valueobject.License) error {
	// 获取新 ID
	id, err := r.db.NextContentId(NsLicense)
	if err != nil {
		return fmt.Errorf("failed to get next ID: %w", err)
	}
	license.ID = int(id)

	license.SetHash()
	license.SetSlug("")

	data, err := json.Marshal(license)
	if err != nil {
		return err
	}
	return r.db.NewContent(license, data)
}

// ========== Device 操作 ==========

// GetDeviceByID 通过 License + DeviceID 查找设备
func (r *LicenseRepository) GetDeviceByID(licenseKey, deviceID string) (*valueobject.LicenseDevice, error) {
	hashKey := hash.MD5(licenseKey + ":" + deviceID)

	idBytes, err := r.db.GetIdByHash(NsLicenseDevice, hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("device not found")
	}

	data, err := r.db.GetContent(NsLicenseDevice, string(idBytes))
	if err != nil {
		return nil, err
	}

	var device valueobject.LicenseDevice
	if err := json.Unmarshal(data, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

// GetDevicesByLicense 获取某 License 的所有设备
func (r *LicenseRepository) GetDevicesByLicense(licenseKey string) ([]valueobject.LicenseDevice, error) {
	prefix := fmt.Sprintf("%s:", licenseKey)

	results, err := r.db.ContentByPrefix(NsLicenseDevice, prefix)
	if err != nil {
		return nil, err
	}

	devices := make([]valueobject.LicenseDevice, 0, len(results))
	for _, data := range results {
		var device valueobject.LicenseDevice
		if err := json.Unmarshal(data, &device); err != nil {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// SaveDevice 保存设备记录
func (r *LicenseRepository) SaveDevice(device *valueobject.LicenseDevice) error {
	if device.ID <= 0 {
		// 新记录 - 初始化 Item 字段
		id, err := r.db.NextContentId(NsLicenseDevice)
		if err != nil {
			return fmt.Errorf("failed to get next ID: %w", err)
		}
		device.ID = int(id)
		device.Namespace = NsLicenseDevice
		device.Status = "public"
		
		// 初始化 UUID
		if device.UUID.String() == "00000000-0000-0000-0000-000000000000" {
			newUUID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("failed to generate UUID: %w", err)
			}
			device.UUID = newUUID
		}
		
		// 初始化时间戳
		if device.Timestamp == 0 {
			device.Timestamp = device.FirstSeenAt
		}
		if device.Updated == 0 {
			device.Updated = device.LastSeenAt
		}
	}
	
	device.SetHash()
	device.SetSlug("")

	data, err := json.Marshal(device)
	if err != nil {
		return err
	}

	if device.ID <= 0 {
		return r.db.NewContent(device, data)
	}
	return r.db.PutContent(device, data)
}

// ========== IP 操作 ==========

// GetIPByAddress 通过 License + IPAddress 查找 IP 记录
func (r *LicenseRepository) GetIPByAddress(licenseKey, ipAddress string) (*valueobject.LicenseIP, error) {
	hashKey := hash.MD5(licenseKey + ":" + ipAddress)

	idBytes, err := r.db.GetIdByHash(NsLicenseIP, hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("IP not found")
	}

	data, err := r.db.GetContent(NsLicenseIP, string(idBytes))
	if err != nil {
		return nil, err
	}

	var ip valueobject.LicenseIP
	if err := json.Unmarshal(data, &ip); err != nil {
		return nil, err
	}

	return &ip, nil
}

// GetIPsByLicense 获取某 License 的所有 IP 记录
func (r *LicenseRepository) GetIPsByLicense(licenseKey string) ([]valueobject.LicenseIP, error) {
	prefix := fmt.Sprintf("%s:", licenseKey)

	results, err := r.db.ContentByPrefix(NsLicenseIP, prefix)
	if err != nil {
		return nil, err
	}

	ips := make([]valueobject.LicenseIP, 0, len(results))
	for _, data := range results {
		var ip valueobject.LicenseIP
		if err := json.Unmarshal(data, &ip); err != nil {
			continue
		}
		ips = append(ips, ip)
	}

	return ips, nil
}

// SaveIP 保存 IP 记录
func (r *LicenseRepository) SaveIP(ip *valueobject.LicenseIP) error {
	if ip.ID <= 0 {
		// 新记录 - 初始化 Item 字段
		id, err := r.db.NextContentId(NsLicenseIP)
		if err != nil {
			return fmt.Errorf("failed to get next ID: %w", err)
		}
		ip.ID = int(id)
		ip.Namespace = NsLicenseIP
		ip.Status = "public"
		
		// 初始化 UUID
		if ip.UUID.String() == "00000000-0000-0000-0000-000000000000" {
			newUUID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("failed to generate UUID: %w", err)
			}
			ip.UUID = newUUID
		}
		
		// 初始化时间戳
		if ip.Timestamp == 0 {
			ip.Timestamp = ip.FirstSeenAt
		}
		if ip.Updated == 0 {
			ip.Updated = ip.LastSeenAt
		}
	}
	
	ip.SetHash()
	ip.SetSlug("")

	data, err := json.Marshal(ip)
	if err != nil {
		return err
	}

	if ip.ID <= 0 {
		return r.db.NewContent(ip, data)
	}
	return r.db.PutContent(ip, data)
}

// ========== SyncAccount 操作 ==========

// GetSyncAccountByLicense 通过 License 查找 SyncAccount
func (r *LicenseRepository) GetSyncAccountByLicense(licenseKey string) (*valueobject.SyncAccount, error) {
	hashKey := hash.MD5(licenseKey)

	idBytes, err := r.db.GetIdByHash(NsSyncAccount, hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("sync account not found")
	}

	data, err := r.db.GetContent(NsSyncAccount, string(idBytes))
	if err != nil {
		return nil, err
	}

	var account valueobject.SyncAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// SaveSyncAccount 保存 SyncAccount
func (r *LicenseRepository) SaveSyncAccount(account *valueobject.SyncAccount) error {
	account.SetHash()
	account.SetSlug("")

	data, err := json.Marshal(account)
	if err != nil {
		return err
	}

	if account.ID <= 0 {
		id, err := r.db.NextContentId(NsSyncAccount)
		if err != nil {
			return fmt.Errorf("failed to get next ID: %w", err)
		}
		account.ID = int(id)
		return r.db.NewContent(account, data)
	}
	return r.db.PutContent(account, data)
}

// SaveSyncUsage 保存 SyncUsage 记录
func (r *LicenseRepository) SaveSyncUsage(usage *valueobject.SyncUsage) error {
	usage.SetHash()
	usage.SetSlug("")

	id, err := r.db.NextContentId(NsSyncUsage)
	if err != nil {
		return fmt.Errorf("failed to get next ID: %w", err)
	}
	usage.ID = int(id)

	data, err := json.Marshal(usage)
	if err != nil {
		return err
	}
	return r.db.NewContent(usage, data)
}

// ========== PublishSite 操作 ==========

// GetPublishSitesByLicense 获取某 License 的所有站点
func (r *LicenseRepository) GetPublishSitesByLicense(licenseKey string) ([]valueobject.PublishSite, error) {
	prefix := fmt.Sprintf("%s:", licenseKey)

	results, err := r.db.ContentByPrefix(NsPublishSite, prefix)
	if err != nil {
		return nil, err
	}

	sites := make([]valueobject.PublishSite, 0, len(results))
	for _, data := range results {
		var site valueobject.PublishSite
		if err := json.Unmarshal(data, &site); err != nil {
			continue
		}
		sites = append(sites, site)
	}

	return sites, nil
}

// SavePublishSite 保存 PublishSite
func (r *LicenseRepository) SavePublishSite(site *valueobject.PublishSite) error {
	site.SetHash()
	site.SetSlug("")

	data, err := json.Marshal(site)
	if err != nil {
		return err
	}

	if site.ID <= 0 {
		id, err := r.db.NextContentId(NsPublishSite)
		if err != nil {
			return fmt.Errorf("failed to get next ID: %w", err)
		}
		site.ID = int(id)
		return r.db.NewContent(site, data)
	}
	return r.db.PutContent(site, data)
}

// SavePublishUsage 保存 PublishUsage 记录
func (r *LicenseRepository) SavePublishUsage(usage *valueobject.PublishUsage) error {
	usage.SetHash()
	usage.SetSlug("")

	id, err := r.db.NextContentId(NsPublishUsage)
	if err != nil {
		return fmt.Errorf("failed to get next ID: %w", err)
	}
	usage.ID = int(id)

	data, err := json.Marshal(usage)
	if err != nil {
		return err
	}
	return r.db.NewContent(usage, data)
}

// ========== PublishDomain 操作 ==========

// SavePublishDomain 保存 PublishDomain
func (r *LicenseRepository) SavePublishDomain(domain *valueobject.PublishDomain) error {
	domain.SetHash()
	domain.SetSlug("")

	data, err := json.Marshal(domain)
	if err != nil {
		return err
	}

	if domain.ID <= 0 {
		id, err := r.db.NextContentId(NsPublishDomain)
		if err != nil {
			return fmt.Errorf("failed to get next ID: %w", err)
		}
		domain.ID = int(id)
		return r.db.NewContent(domain, data)
	}
	return r.db.PutContent(domain, data)
}

// GetPublishSiteByName 通过名称获取站点
func (r *LicenseRepository) GetPublishSiteByName(licenseKey, name string) (*valueobject.PublishSite, error) {
	hashKey := hash.MD5(licenseKey + ":" + name)

	idBytes, err := r.db.GetIdByHash(NsPublishSite, hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("site not found: %s", name)
	}

	data, err := r.db.GetContent(NsPublishSite, string(idBytes))
	if err != nil {
		return nil, err
	}

	var site valueobject.PublishSite
	if err := json.Unmarshal(data, &site); err != nil {
		return nil, err
	}

	return &site, nil
}

// DeletePublishSite 删除站点
func (r *LicenseRepository) DeletePublishSite(licenseKey, name string) error {
	site, err := r.GetPublishSiteByName(licenseKey, name)
	if err != nil {
		return err
	}

	slug := fmt.Sprintf("%s:%s", licenseKey, name)
	return r.db.DeleteContent(NsPublishSite, fmt.Sprintf("%d", site.ID), slug, site.Hash)
}

// GetPublishUsageByLicense 获取最新的使用量记录
func (r *LicenseRepository) GetPublishUsageByLicense(licenseKey string) (*valueobject.PublishUsage, error) {
	// 获取所有使用量记录
	prefix := fmt.Sprintf("%s:", licenseKey)
	results, err := r.db.ContentByPrefix(NsPublishUsage, prefix)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("usage not found")
	}

	// 返回最新的记录
	var latestUsage *valueobject.PublishUsage
	var latestTime int64
	for _, data := range results {
		var usage valueobject.PublishUsage
		if err := json.Unmarshal(data, &usage); err != nil {
			continue
		}
		if usage.RecordedAt > latestTime {
			latestTime = usage.RecordedAt
			latestUsage = &usage
		}
	}

	if latestUsage == nil {
		return nil, fmt.Errorf("usage not found")
	}
	return latestUsage, nil
}

// GetPublishDomainBySite 通过站点获取域名
func (r *LicenseRepository) GetPublishDomainBySite(licenseKey, siteName string) (*valueobject.PublishDomain, error) {
	// 使用前缀查找
	prefix := fmt.Sprintf("%s:", licenseKey)
	results, err := r.db.ContentByPrefix(NsPublishDomain, prefix)
	if err != nil {
		return nil, err
	}

	for _, data := range results {
		var domain valueobject.PublishDomain
		if err := json.Unmarshal(data, &domain); err != nil {
			continue
		}
		if domain.PublishSite == siteName {
			return &domain, nil
		}
	}

	return nil, fmt.Errorf("domain not found for site: %s", siteName)
}

// GetPublishDomainsByLicense 获取 License 的所有域名
func (r *LicenseRepository) GetPublishDomainsByLicense(licenseKey string) ([]valueobject.PublishDomain, error) {
	prefix := fmt.Sprintf("%s:", licenseKey)
	results, err := r.db.ContentByPrefix(NsPublishDomain, prefix)
	if err != nil {
		return nil, err
	}

	domains := make([]valueobject.PublishDomain, 0, len(results))
	for _, data := range results {
		var domain valueobject.PublishDomain
		if err := json.Unmarshal(data, &domain); err != nil {
			continue
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// DeletePublishDomain 删除域名
func (r *LicenseRepository) DeletePublishDomain(licenseKey, domainName string) error {
	hashKey := hash.MD5(licenseKey + ":" + domainName)

	idBytes, err := r.db.GetIdByHash(NsPublishDomain, hashKey)
	if err != nil || idBytes == nil {
		return nil // 域名不存在，忽略
	}

	slug := fmt.Sprintf("%s:%s", licenseKey, domainName)
	return r.db.DeleteContent(NsPublishDomain, string(idBytes), slug, hashKey)
}

// GetAllActivePublishDomains 获取所有活跃的域名
func (r *LicenseRepository) GetAllActivePublishDomains() ([]valueobject.PublishDomain, error) {
	results, err := r.db.ContentByPrefix(NsPublishDomain, "")
	if err != nil {
		return nil, err
	}

	domains := make([]valueobject.PublishDomain, 0)
	for _, data := range results {
		var domain valueobject.PublishDomain
		if err := json.Unmarshal(data, &domain); err != nil {
			continue
		}
		if domain.Status == "active" {
			domains = append(domains, domain)
		}
	}

	return domains, nil
}

// 确保实现 sync 和 publish 的 Repository 接口
var _ syncEntity.Repository = (*LicenseRepository)(nil)
var _ publishEntity.Repository = (*LicenseRepository)(nil)

