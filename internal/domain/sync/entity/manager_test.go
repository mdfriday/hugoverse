package entity

import (
	"fmt"
	"testing"
	"time"

	adminVO "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// MockCouchDBClient 模拟 CouchDB 客户端
type MockCouchDBClient struct {
	databases   map[string]bool
	users       map[string]string
	permissions map[string]string
}

func NewMockCouchDBClient() *MockCouchDBClient {
	return &MockCouchDBClient{
		databases:   make(map[string]bool),
		users:       make(map[string]string),
		permissions: make(map[string]string),
	}
}

func (m *MockCouchDBClient) CreateDatabase(name string) error {
	m.databases[name] = true
	return nil
}

func (m *MockCouchDBClient) CreateUser(email, password string) error {
	m.users[email] = password
	return nil
}

func (m *MockCouchDBClient) SetDatabasePermission(dbName, email string) error {
	m.permissions[dbName] = email
	return nil
}

func (m *MockCouchDBClient) GetDatabaseInfo(name string) (*DatabaseInfo, error) {
	if !m.databases[name] {
		return nil, fmt.Errorf("database not found")
	}
	return &DatabaseInfo{
		DocCount: 10,
		DiskSize: 1024 * 1024, // 1MB
	}, nil
}

// MockRepository 模拟数据仓库
type MockRepository struct {
	licenses     map[string]*contentVO.License
	devices      map[string]*contentVO.LicenseDevice
	ips          map[string]*contentVO.LicenseIP
	syncAccounts map[string]*contentVO.SyncAccount
	syncUsages   []*contentVO.SyncUsage
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		licenses:     make(map[string]*contentVO.License),
		devices:      make(map[string]*contentVO.LicenseDevice),
		ips:          make(map[string]*contentVO.LicenseIP),
		syncAccounts: make(map[string]*contentVO.SyncAccount),
		syncUsages:   make([]*contentVO.SyncUsage, 0),
	}
}

func (r *MockRepository) GetLicenseByKey(key string) (*contentVO.License, error) {
	if l, ok := r.licenses[key]; ok {
		return l, nil
	}
	return nil, fmt.Errorf("license not found")
}

func (r *MockRepository) UpdateLicense(license *contentVO.License) error {
	r.licenses[license.LicenseKey] = license
	return nil
}

func (r *MockRepository) GetDevicesByLicense(licenseKey string) ([]contentVO.LicenseDevice, error) {
	var devices []contentVO.LicenseDevice
	for _, d := range r.devices {
		if d.License == licenseKey {
			devices = append(devices, *d)
		}
	}
	return devices, nil
}

func (r *MockRepository) GetIPsByLicense(licenseKey string) ([]contentVO.LicenseIP, error) {
	var ips []contentVO.LicenseIP
	for _, ip := range r.ips {
		if ip.License == licenseKey {
			ips = append(ips, *ip)
		}
	}
	return ips, nil
}

func (r *MockRepository) SaveDevice(device *contentVO.LicenseDevice) error {
	key := device.License + ":" + device.DeviceID
	r.devices[key] = device
	return nil
}

func (r *MockRepository) SaveIP(ip *contentVO.LicenseIP) error {
	key := ip.License + ":" + ip.IPAddress
	r.ips[key] = ip
	return nil
}

func (r *MockRepository) GetDeviceByID(licenseKey, deviceID string) (*contentVO.LicenseDevice, error) {
	key := licenseKey + ":" + deviceID
	if d, ok := r.devices[key]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("device not found")
}

func (r *MockRepository) GetIPByAddress(licenseKey, ipAddress string) (*contentVO.LicenseIP, error) {
	key := licenseKey + ":" + ipAddress
	if ip, ok := r.ips[key]; ok {
		return ip, nil
	}
	return nil, fmt.Errorf("IP not found")
}

func (r *MockRepository) SaveSyncAccount(account *contentVO.SyncAccount) error {
	r.syncAccounts[account.License] = account
	return nil
}

func (r *MockRepository) GetSyncAccountByLicense(licenseKey string) (*contentVO.SyncAccount, error) {
	if a, ok := r.syncAccounts[licenseKey]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("sync account not found")
}

func (r *MockRepository) SaveSyncUsage(usage *contentVO.SyncUsage) error {
	r.syncUsages = append(r.syncUsages, usage)
	return nil
}

// ========== 测试 ==========

func TestManagerValidateAndRecordAccess(t *testing.T) {
	config := &adminVO.CouchDBConfig{
		URL:      "http://localhost:5984",
		DBPrefix: "userdb-",
	}
	client := NewMockCouchDBClient()
	repo := NewMockRepository()

	// 添加有效的 License
	license := &contentVO.License{
		LicenseKey:     "MDF-TEST-1234-5678",
		Plan:           contentVO.PlanStarter,
		Activated:      true,
		ExpiryDate:     time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
		MaxDevices:     3,
		MaxIPs:         3,
		CurrentDevices: 0,
		CurrentIPs:     0,
	}
	repo.licenses[license.LicenseKey] = license

	manager := NewManager(config, client, repo)

	// 测试第一次访问
	err := manager.ValidateAndRecordAccess(
		"MDF-TEST-1234-5678",
		"device-001",
		"MacBook Pro",
		"desktop",
		"192.168.1.100",
	)
	if err != nil {
		t.Errorf("ValidateAndRecordAccess() error = %v", err)
	}

	// 验证设备被记录
	devices, _ := repo.GetDevicesByLicense("MDF-TEST-1234-5678")
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	// 验证 IP 被记录
	ips, _ := repo.GetIPsByLicense("MDF-TEST-1234-5678")
	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}

	// 验证 License 计数更新
	updatedLicense := repo.licenses["MDF-TEST-1234-5678"]
	if updatedLicense.CurrentDevices != 1 {
		t.Errorf("Expected CurrentDevices = 1, got %d", updatedLicense.CurrentDevices)
	}
	if updatedLicense.CurrentIPs != 1 {
		t.Errorf("Expected CurrentIPs = 1, got %d", updatedLicense.CurrentIPs)
	}
}

func TestManagerDeviceLimit(t *testing.T) {
	config := &adminVO.CouchDBConfig{}
	client := NewMockCouchDBClient()
	repo := NewMockRepository()

	// 添加已达到设备限制的 License
	license := &contentVO.License{
		LicenseKey:     "MDF-TEST-LIMIT",
		Plan:           contentVO.PlanStarter,
		Activated:      true,
		ExpiryDate:     time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
		MaxDevices:     1,
		MaxIPs:         3,
		CurrentDevices: 1,
		CurrentIPs:     0,
	}
	repo.licenses[license.LicenseKey] = license

	manager := NewManager(config, client, repo)

	// 尝试添加新设备应该失败
	err := manager.ValidateAndRecordAccess(
		"MDF-TEST-LIMIT",
		"new-device",
		"iPhone",
		"mobile",
		"192.168.1.100",
	)
	if err == nil {
		t.Error("Expected error for device limit, got nil")
	}
}

func TestManagerCreateSyncAccount(t *testing.T) {
	config := &adminVO.CouchDBConfig{
		URL:      "http://localhost:5984",
		DBPrefix: "userdb-",
	}
	client := NewMockCouchDBClient()
	repo := NewMockRepository()

	license := &contentVO.License{
		LicenseKey: "MDF-SYNC-TEST-1234",
		Plan:       contentVO.PlanStarter, // Starter 支持 Sync
		Activated:  true,
		ExpiryDate: time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	manager := NewManager(config, client, repo)

	// 创建 Sync 账号
	account, err := manager.CreateSyncAccount(license)
	if err != nil {
		t.Errorf("CreateSyncAccount() error = %v", err)
	}

	if account == nil {
		t.Fatal("Expected account, got nil")
	}

	// 验证账号信息
	if account.Email != license.ToEmail() {
		t.Errorf("Email = %v, want %v", account.Email, license.ToEmail())
	}

	// 验证数据库已创建
	expectedDBName := "userdb-" + license.ToUserDir()
	if !client.databases[expectedDBName] {
		t.Errorf("Database %s was not created", expectedDBName)
	}

	// 验证用户已创建
	if _, ok := client.users[license.ToEmail()]; !ok {
		t.Errorf("User %s was not created", license.ToEmail())
	}
}

func TestManagerCreateSyncAccountFreePlan(t *testing.T) {
	config := &adminVO.CouchDBConfig{}
	client := NewMockCouchDBClient()
	repo := NewMockRepository()

	license := &contentVO.License{
		LicenseKey: "MDF-FREE-TEST-1234",
		Plan:       contentVO.PlanFree, // Free 不支持 Sync
		Activated:  true,
		ExpiryDate: time.Now().Add(7 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	manager := NewManager(config, client, repo)

	// 尝试创建 Sync 账号应该失败
	_, err := manager.CreateSyncAccount(license)
	if err == nil {
		t.Error("Expected error for Free plan, got nil")
	}
}

func TestManagerUpdateUsage(t *testing.T) {
	config := &adminVO.CouchDBConfig{
		URL:      "http://localhost:5984",
		DBPrefix: "userdb-",
	}
	client := NewMockCouchDBClient()
	repo := NewMockRepository()

	license := &contentVO.License{
		LicenseKey: "MDF-USAGE-TEST",
		Plan:       contentVO.PlanStarter,
		Activated:  true,
		ExpiryDate: time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}
	repo.licenses[license.LicenseKey] = license

	// 先创建账号
	dbName := "userdb-" + license.ToUserDir()
	client.databases[dbName] = true

	repo.syncAccounts[license.LicenseKey] = &contentVO.SyncAccount{
		License: license.LicenseKey,
		DBName:  dbName,
	}

	manager := NewManager(config, client, repo)

	// 更新使用量
	usage, err := manager.UpdateUsage(license.LicenseKey)
	if err != nil {
		t.Errorf("UpdateUsage() error = %v", err)
	}

	if usage == nil {
		t.Fatal("Expected usage, got nil")
	}

	if usage.DocumentCount != 10 {
		t.Errorf("DocumentCount = %d, want 10", usage.DocumentCount)
	}

	if usage.StorageBytes != 1024*1024 {
		t.Errorf("StorageBytes = %d, want %d", usage.StorageBytes, 1024*1024)
	}
}

