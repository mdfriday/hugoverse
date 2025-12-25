package entity

import (
	"fmt"
	"time"

	adminVO "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// Manager Sync 业务逻辑管理器
// 负责设备/IP 验证、CouchDB 账号分配、用量监控
type Manager struct {
	config      *adminVO.CouchDBConfig
	couchClient CouchDBClient
	repo        Repository
}

// CouchDBClient CouchDB 操作接口
type CouchDBClient interface {
	CreateDatabase(name string) error
	CreateUser(email, password string) error
	SetDatabasePermission(dbName, email string) error
	GetDatabaseInfo(name string) (*DatabaseInfo, error)
}

// DatabaseInfo CouchDB 数据库信息
type DatabaseInfo struct {
	DocCount int   `json:"doc_count"`
	DiskSize int64 `json:"disk_size"`
}

// Repository 数据仓库接口
type Repository interface {
	// License
	GetLicenseByKey(key string) (*contentVO.License, error)
	UpdateLicense(license *contentVO.License) error

	// 设备/IP
	GetDevicesByLicense(licenseKey string) ([]contentVO.LicenseDevice, error)
	GetIPsByLicense(licenseKey string) ([]contentVO.LicenseIP, error)
	SaveDevice(device *contentVO.LicenseDevice) error
	SaveIP(ip *contentVO.LicenseIP) error
	GetDeviceByID(licenseKey, deviceID string) (*contentVO.LicenseDevice, error)
	GetIPByAddress(licenseKey, ipAddress string) (*contentVO.LicenseIP, error)

	// Sync
	SaveSyncAccount(account *contentVO.SyncAccount) error
	GetSyncAccountByLicense(licenseKey string) (*contentVO.SyncAccount, error)
	SaveSyncUsage(usage *contentVO.SyncUsage) error
}

// NewManager 创建 Sync Manager
func NewManager(config *adminVO.CouchDBConfig, client CouchDBClient, repo Repository) *Manager {
	return &Manager{
		config:      config,
		couchClient: client,
		repo:        repo,
	}
}

// ========== 设备/IP 验证 (治理逻辑) ==========

// ValidateAndRecordAccess 验证设备和 IP，记录访问
func (m *Manager) ValidateAndRecordAccess(licenseKey, deviceID, deviceName, deviceType, ipAddress string) error {
	license, err := m.repo.GetLicenseByKey(licenseKey)
	if err != nil {
		return fmt.Errorf("license not found: %w", err)
	}

	if !license.IsValid() {
		return fmt.Errorf("license is not valid or expired")
	}

	// 检查设备
	if err := m.checkAndRecordDevice(license, deviceID, deviceName, deviceType); err != nil {
		return err
	}

	// 检查 IP
	if err := m.checkAndRecordIP(license, ipAddress); err != nil {
		return err
	}

	return nil
}

func (m *Manager) checkAndRecordDevice(license *contentVO.License, deviceID, deviceName, deviceType string) error {
	now := time.Now().UnixMilli()

	// 查找已存在的设备
	existingDevice, err := m.repo.GetDeviceByID(license.LicenseKey, deviceID)
	if err == nil && existingDevice != nil {
		// 设备已存在，更新访问记录
		existingDevice.LastSeenAt = now
		existingDevice.AccessCount++
		return m.repo.SaveDevice(existingDevice)
	}

	// 新设备 - 检查限制
	if !license.CanAddDevice() {
		return fmt.Errorf("device limit reached (%d/%d)", license.CurrentDevices, license.MaxDevices)
	}

	// 创建新设备记录
	device := &contentVO.LicenseDevice{
		License:     license.LicenseKey,
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		DeviceType:  deviceType,
		FirstSeenAt: now,
		LastSeenAt:  now,
		AccessCount: 1,
		Status:      "active",
	}

	if err := m.repo.SaveDevice(device); err != nil {
		return fmt.Errorf("failed to save device: %w", err)
	}

	// 更新 License 设备计数
	license.CurrentDevices++
	return m.repo.UpdateLicense(license)
}

func (m *Manager) checkAndRecordIP(license *contentVO.License, ipAddress string) error {
	now := time.Now().UnixMilli()

	// 查找已存在的 IP
	existingIP, err := m.repo.GetIPByAddress(license.LicenseKey, ipAddress)
	if err == nil && existingIP != nil {
		// IP 已存在，更新访问记录
		existingIP.LastSeenAt = now
		existingIP.AccessCount++
		return m.repo.SaveIP(existingIP)
	}

	// 新 IP - 检查限制
	if !license.CanAddIP() {
		return fmt.Errorf("IP limit reached (%d/%d)", license.CurrentIPs, license.MaxIPs)
	}

	// 创建新 IP 记录
	ip := &contentVO.LicenseIP{
		License:     license.LicenseKey,
		IPAddress:   ipAddress,
		FirstSeenAt: now,
		LastSeenAt:  now,
		AccessCount: 1,
		Status:      "active",
	}

	if err := m.repo.SaveIP(ip); err != nil {
		return fmt.Errorf("failed to save IP: %w", err)
	}

	// 更新 License IP 计数
	license.CurrentIPs++
	return m.repo.UpdateLicense(license)
}

// ========== CouchDB 账号分配 ==========

// CreateSyncAccount 为 License 创建 CouchDB 同步账号
func (m *Manager) CreateSyncAccount(license *contentVO.License) (*contentVO.SyncAccount, error) {
	// 检查是否已存在
	existing, _ := m.repo.GetSyncAccountByLicense(license.LicenseKey)
	if existing != nil {
		return existing, nil
	}

	// 检查 License 是否支持 Sync
	features := license.GetFeatures()
	if !features.SyncEnabled {
		return nil, fmt.Errorf("sync not enabled for plan: %s", license.Plan)
	}

	email := license.ToEmail()
	password := license.ToPassword()
	dbName := fmt.Sprintf("%s%s", m.config.DBPrefix, license.ToUserDir())

	// 创建 CouchDB 数据库
	if err := m.couchClient.CreateDatabase(dbName); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// 创建用户
	if err := m.couchClient.CreateUser(email, password); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 设置数据库权限
	if err := m.couchClient.SetDatabasePermission(dbName, email); err != nil {
		return nil, fmt.Errorf("failed to set database permission: %w", err)
	}

	account := &contentVO.SyncAccount{
		License:    license.LicenseKey,
		Email:      email,
		DBName:     dbName,
		DBEndpoint: fmt.Sprintf("%s/%s", m.config.URL, dbName),
		Status:     "active",
		CreatedAt:  time.Now().UnixMilli(),
	}

	if err := m.repo.SaveSyncAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save sync account: %w", err)
	}

	return account, nil
}

// GetSyncAccount 获取 License 的 Sync 账号信息
func (m *Manager) GetSyncAccount(licenseKey string) (*contentVO.SyncAccount, error) {
	return m.repo.GetSyncAccountByLicense(licenseKey)
}

// ========== 使用量监控 ==========

// UpdateUsage 更新并记录 Sync 使用量
func (m *Manager) UpdateUsage(licenseKey string) (*contentVO.SyncUsage, error) {
	account, err := m.repo.GetSyncAccountByLicense(licenseKey)
	if err != nil {
		return nil, fmt.Errorf("sync account not found: %w", err)
	}

	// 从 CouchDB 获取数据库信息
	dbInfo, err := m.couchClient.GetDatabaseInfo(account.DBName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	// 获取 License 配额
	license, err := m.repo.GetLicenseByKey(licenseKey)
	if err != nil {
		return nil, err
	}
	features := license.GetFeatures()

	now := time.Now().UnixMilli()
	usage := &contentVO.SyncUsage{
		SyncAccount:   account.License,
		DocumentCount: dbInfo.DocCount,
		StorageBytes:  dbInfo.DiskSize,
		QuotaBytes:    int64(features.SyncQuotaMB) * 1024 * 1024,
		LastSyncAt:    now,
		RecordedAt:    now,
	}

	if err := m.repo.SaveSyncUsage(usage); err != nil {
		return nil, fmt.Errorf("failed to save usage: %w", err)
	}

	return usage, nil
}

// GetDevices 获取 License 的所有设备
func (m *Manager) GetDevices(licenseKey string) ([]contentVO.LicenseDevice, error) {
	return m.repo.GetDevicesByLicense(licenseKey)
}

// GetIPs 获取 License 的所有 IP
func (m *Manager) GetIPs(licenseKey string) ([]contentVO.LicenseIP, error) {
	return m.repo.GetIPsByLicense(licenseKey)
}

// BlockDevice 封禁设备
func (m *Manager) BlockDevice(licenseKey, deviceID string) error {
	device, err := m.repo.GetDeviceByID(licenseKey, deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	device.Status = "blocked"
	return m.repo.SaveDevice(device)
}

// BlockIP 封禁 IP
func (m *Manager) BlockIP(licenseKey, ipAddress string) error {
	ip, err := m.repo.GetIPByAddress(licenseKey, ipAddress)
	if err != nil {
		return fmt.Errorf("IP not found: %w", err)
	}

	ip.Status = "blocked"
	return m.repo.SaveIP(ip)
}

