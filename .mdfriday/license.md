# License 管理系统需求与实现计划

## 1. 需求概述

实现 license 管理系统，可以支持用户创建 sync 服务和 publish 服务。

用户可以在前端激活 license ，就可以知道 license 的类型和有效期。
license 有 Free，Starter，Creator，Pro, Enterprise 不同的类型，有效期通常都是一年。

用户的使用流程是，在前端输入 license , 会发送一个 activate 的请求到后端，验证是否合法，有没有过期， 如果是合法有效的，则会在后端创建一个 couchDB 的数据库。
用户就可以用这个账号进行文件同步了。
同时，我们也会为用户生成一个专属的 WEB SERVER 文件夹，里面可以放用户分享的单篇文章，或者是整个站点，而且每一个都是以独立文件夹存在的，所以我们还可以绑定自定义域名。

**设备/IP 限制**: 每个 License 默认允许 3 台设备、3 个 IP 使用，需要记录使用的设备和 IP 信息，为以后治理所用。

---

## 2. 核心设计

### 2.1 License 转用户机制

每个 License 对应一个虚拟用户账号：

```
License Key: MDF-ABCD-EFGH-JKLM

生成的用户信息:
├── Email: abcd-efgh-jklm@mdfriday.com
├── Password: base64("abcd-efgh-jklm") = "YWJjZC1lZmdoLWprbG0="
└── UserDir: hash(email)[:16]
```

这样每个 License 就像一个独立用户，可以：
- 拥有独立的 CouchDB 数据库 (用于 Sync)
- 拥有独立的文件目录 (用于 Publish)

### 2.2 领域划分

```
internal/
├── application/
│   └── dir.go                   # 添加 PublishDir() 函数
├── domain/
│   ├── content/valueobject/
│   │   ├── license.go           # License 数据结构 (内容资源)
│   │   ├── licensedevice.go     # 设备记录表 (治理用)
│   │   ├── licenseip.go         # IP 记录表 (治理用)
│   │   ├── syncaccount.go       # Sync 账号关系表
│   │   ├── syncusage.go         # Sync 使用量记录
│   │   ├── publishsite.go       # Publish 站点记录
│   │   ├── publishusage.go      # Publish 容量记录
│   │   └── publishdomain.go     # 自定义域名记录
│   ├── admin/valueobject/
│   │   └── couchdb.go           # CouchDB 配置 (仅配置)
│   ├── sync/                    # Sync Domain - 业务逻辑
│   │   └── entity/manager.go    # 设备/IP 验证、CouchDB 账号分配、用量监控
│   └── publish/                 # Publish Domain - 业务逻辑
│       └── entity/manager.go    # 用量管理、自定义域名
└── interfaces/api/handler/
    └── ...
```

### 2.3 参考现有实现

```go
// 现有实现
func PreviewDir() string {
    return filepath.Join(DataDir(), "preview")
}

// 新增 - Publish 目录
func PublishDir() string {
    return filepath.Join(DataDir(), "publish")
}
```

---

## 3. 任务拆解

### 任务 1: Content Domain - License ValueObject ⭐ 优先级: 高

**文件**: `internal/domain/content/valueobject/license.go`

```go
package valueobject

import (
    "encoding/base64"
    "fmt"
    "strings"
    "time"
    
    "github.com/mdfriday/hugoverse/pkg/editor"
    "github.com/mdfriday/hugoverse/pkg/hash"
)

type LicensePlan string

const (
    PlanFree       LicensePlan = "free"
    PlanStarter    LicensePlan = "starter"
    PlanCreator    LicensePlan = "creator"
    PlanPro        LicensePlan = "pro"
    PlanEnterprise LicensePlan = "enterprise"
)

// License 作为 Content ValueObject
type License struct {
    Item

    LicenseKey string      `json:"license_key"`  // MDF-XXXX-XXXX-XXXX
    Plan       LicensePlan `json:"plan"`
    
    // 有效期
    IssueDate   int64 `json:"issue_date"`
    ExpiryDate  int64 `json:"expiry_date"`
    
    // 激活状态
    Activated   bool  `json:"activated"`
    ActivatedAt int64 `json:"activated_at"`
    
    // 设备/IP 限制 (治理用)
    MaxDevices     int `json:"max_devices"`      // 最大设备数，默认 3
    MaxIPs         int `json:"max_ips"`          // 最大 IP 数，默认 3
    CurrentDevices int `json:"current_devices"`  // 当前设备数
    CurrentIPs     int `json:"current_ips"`      // 当前 IP 数
}

func (l *License) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(l,
        editor.Field{
            View: editor.Input("LicenseKey", l, map[string]string{
                "label": "License Key",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Select("Plan", l, map[string]string{
                "label": "Plan",
            }, map[string]string{
                "free": "Free", "starter": "Starter", 
                "creator": "Creator", "pro": "Pro", "enterprise": "Enterprise",
            }),
        },
        editor.Field{
            View: editor.Input("MaxDevices", l, map[string]string{
                "label": "Max Devices",
                "type":  "number",
            }),
        },
        editor.Field{
            View: editor.Input("MaxIPs", l, map[string]string{
                "label": "Max IPs",
                "type":  "number",
            }),
        },
    )
    return view, err
}

func (l *License) String() string {
    return fmt.Sprintf("%s (%s)", l.LicenseKey, l.Plan)
}

func (l *License) SetHash() {
    l.Hash = hash.MD5(l.LicenseKey)
}

func (l *License) IndexContent() bool { return true }

// ========== License 转用户机制 ==========

func (l *License) ToEmail() string {
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return fmt.Sprintf("%s@mdfriday.com", key)
}

func (l *License) ToPassword() string {
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return base64.StdEncoding.EncodeToString([]byte(key))
}

func (l *License) ToUserDir() string {
    return hash.MD5(l.ToEmail())[:16]
}

func (l *License) IsExpired() bool {
    return time.Now().UnixMilli() > l.ExpiryDate
}

func (l *License) IsValid() bool {
    return l.Activated && !l.IsExpired()
}

func (l *License) GetFeatures() *LicenseFeatures {
    return GetPlanFeatures(l.Plan)
}

// ========== 设备/IP 限制检查 ==========

func (l *License) CanAddDevice() bool {
    return l.CurrentDevices < l.MaxDevices
}

func (l *License) CanAddIP() bool {
    return l.CurrentIPs < l.MaxIPs
}
```

**权限定义**:
```go
type LicenseFeatures struct {
    // 设备/IP 限制
    MaxDevices   int  `json:"max_devices"`
    MaxIPs       int  `json:"max_ips"`
    
    // Sync 功能
    SyncEnabled  bool `json:"sync_enabled"`
    SyncQuotaMB  int  `json:"sync_quota"`
    
    // Publish 功能
    PublishEnabled bool `json:"publish_enabled"`
    MaxSites       int  `json:"max_sites"`
    MaxStorageMB   int  `json:"max_storage"`
    CustomDomain   bool `json:"custom_domain"`
}

func GetPlanFeatures(plan LicensePlan) *LicenseFeatures {
    switch plan {
    case PlanFree:
        return &LicenseFeatures{
            MaxDevices: 1, MaxIPs: 1,
            SyncEnabled: false, SyncQuotaMB: 0,
            PublishEnabled: false, MaxSites: 0, MaxStorageMB: 100, CustomDomain: false,
        }
    case PlanStarter:
        return &LicenseFeatures{
            MaxDevices: 3, MaxIPs: 3,
            SyncEnabled: true, SyncQuotaMB: 500,
            PublishEnabled: true, MaxSites: 3, MaxStorageMB: 1024, CustomDomain: false,
        }
    case PlanCreator:
        return &LicenseFeatures{
            MaxDevices: 5, MaxIPs: 5,
            SyncEnabled: true, SyncQuotaMB: 2048,
            PublishEnabled: true, MaxSites: 10, MaxStorageMB: 5120, CustomDomain: true,
        }
    case PlanPro:
        return &LicenseFeatures{
            MaxDevices: 10, MaxIPs: 10,
            SyncEnabled: true, SyncQuotaMB: 10240,
            PublishEnabled: true, MaxSites: 50, MaxStorageMB: 20480, CustomDomain: true,
        }
    case PlanEnterprise:
        return &LicenseFeatures{
            MaxDevices: -1, MaxIPs: -1, // 无限制
            SyncEnabled: true, SyncQuotaMB: 51200,
            PublishEnabled: true, MaxSites: -1, MaxStorageMB: 102400, CustomDomain: true,
        }
    default:
        return &LicenseFeatures{}
    }
}
```

---

### 任务 2: Content Domain - 设备/IP 记录表 ⭐ 优先级: 高

**文件**: `internal/domain/content/valueobject/licensedevice.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// LicenseDevice 设备记录表 (治理用)
type LicenseDevice struct {
    Item

    License    string `json:"license"`      // 关联的 License
    DeviceID   string `json:"device_id"`    // 设备唯一标识
    DeviceName string `json:"device_name"`  // 设备名称 (UA/OS 等)
    DeviceType string `json:"device_type"`  // desktop / mobile / tablet
    
    // 使用信息
    FirstSeenAt int64 `json:"first_seen_at"` // 首次使用时间
    LastSeenAt  int64 `json:"last_seen_at"`  // 最后使用时间
    AccessCount int   `json:"access_count"`  // 访问次数
    
    // 状态
    Status string `json:"status"` // active / blocked
}

func (d *LicenseDevice) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(d,
        editor.Field{
            View: editor.Input("DeviceID", d, map[string]string{
                "label": "Device ID",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Input("DeviceName", d, map[string]string{
                "label": "Device Name",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Select("Status", d, map[string]string{
                "label": "Status",
            }, map[string]string{
                "active":  "Active",
                "blocked": "Blocked",
            }),
        },
    )
    return view, err
}

func (d *LicenseDevice) String() string {
    return fmt.Sprintf("%s - %s", d.DeviceID[:8], d.DeviceName)
}

func (d *LicenseDevice) IndexContent() bool { return true }
```

**文件**: `internal/domain/content/valueobject/licenseip.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// LicenseIP IP 记录表 (治理用)
type LicenseIP struct {
    Item

    License   string `json:"license"`    // 关联的 License
    IPAddress string `json:"ip_address"` // IP 地址
    
    // 地理位置信息 (可选)
    Country string `json:"country"`
    Region  string `json:"region"`
    City    string `json:"city"`
    
    // 使用信息
    FirstSeenAt int64 `json:"first_seen_at"`
    LastSeenAt  int64 `json:"last_seen_at"`
    AccessCount int   `json:"access_count"`
    
    // 状态
    Status string `json:"status"` // active / blocked
}

func (i *LicenseIP) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(i,
        editor.Field{
            View: editor.Input("IPAddress", i, map[string]string{
                "label": "IP Address",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Input("Country", i, map[string]string{
                "label": "Country",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Select("Status", i, map[string]string{
                "label": "Status",
            }, map[string]string{
                "active":  "Active",
                "blocked": "Blocked",
            }),
        },
    )
    return view, err
}

func (i *LicenseIP) String() string {
    return fmt.Sprintf("%s (%s)", i.IPAddress, i.Country)
}

func (i *LicenseIP) IndexContent() bool { return true }
```

---

### 任务 3: Content Domain - Sync 关系数据表 ⭐ 优先级: 中

**文件**: `internal/domain/content/valueobject/syncaccount.go`

```go
package valueobject

// SyncAccount License 对应的 CouchDB 账号
type SyncAccount struct {
    Item

    License    string `json:"license"`
    Email      string `json:"email"`
    DBName     string `json:"db_name"`
    DBEndpoint string `json:"db_endpoint"`
    Status     string `json:"status"` // active / suspended
    CreatedAt  int64  `json:"created_at"`
}

func (s *SyncAccount) String() string {
    return fmt.Sprintf("%s - %s", s.Email, s.DBName)
}

func (s *SyncAccount) IndexContent() bool { return true }
```

**文件**: `internal/domain/content/valueobject/syncusage.go`

```go
package valueobject

// SyncUsage Sync 使用量记录
type SyncUsage struct {
    Item

    SyncAccount   string `json:"sync_account"`
    DocumentCount int    `json:"document_count"`
    StorageBytes  int64  `json:"storage_bytes"`
    QuotaBytes    int64  `json:"quota_bytes"`
    LastSyncAt    int64  `json:"last_sync_at"`
    RecordedAt    int64  `json:"recorded_at"`
}
```

---

### 任务 4: Content Domain - Publish 关系数据表 ⭐ 优先级: 中

**文件**: `internal/domain/content/valueobject/publishsite.go`

```go
package valueobject

// PublishSite 发布站点记录
type PublishSite struct {
    Item

    License      string `json:"license"`
    Name         string `json:"name"`
    SiteType     string `json:"site_type"`  // site / article
    Asset        string `json:"asset"`
    Size         string `json:"size"`
    FolderPath   string `json:"folder_path"`
    PublicURL    string `json:"public_url"`
    Status       string `json:"status"` // pending / active / deleted
    CreatedAt    int64  `json:"created_at"`
}

func (p *PublishSite) String() string { return p.Name }
func (p *PublishSite) Deploy() bool   { return true }
func (p *PublishSite) IndexContent() bool { return true }

func (p *PublishSite) AbsAssetPath(uploadDir string) (string, error) {
    return getAssetAbsPath(p.Asset, uploadDir)
}
```

**文件**: `internal/domain/content/valueobject/publishusage.go`

```go
package valueobject

// PublishUsage Publish 容量记录
type PublishUsage struct {
    Item

    License      string `json:"license"`
    SiteCount    int    `json:"site_count"`
    StorageBytes int64  `json:"storage_bytes"`
    MaxSites     int    `json:"max_sites"`
    QuotaBytes   int64  `json:"quota_bytes"`
    RecordedAt   int64  `json:"recorded_at"`
}
```

**文件**: `internal/domain/content/valueobject/publishdomain.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// PublishDomain 自定义域名记录
// SSL 证书由 Caddy 自动管理和续签
type PublishDomain struct {
    Item

    License     string `json:"license"`      // 关联的 License
    PublishSite string `json:"publish_site"` // 关联的站点
    Domain      string `json:"domain"`       // 自定义域名 (如 blog.example.com)
    TargetPath  string `json:"target_path"`  // 指向的发布目录路径
    
    Status    string `json:"status"`     // active / inactive
    CreatedAt int64  `json:"created_at"`
    UpdatedAt int64  `json:"updated_at"`
}

func (d *PublishDomain) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(d,
        editor.Field{
            View: editor.Input("Domain", d, map[string]string{
                "label":       "Custom Domain",
                "type":        "text",
                "placeholder": "blog.example.com",
            }),
        },
        editor.Field{
            View: editor.Select("Status", d, map[string]string{
                "label": "Status",
            }, map[string]string{
                "active":   "Active",
                "inactive": "Inactive",
            }),
        },
    )
    return view, err
}

func (d *PublishDomain) String() string {
    return d.Domain
}

func (d *PublishDomain) IndexContent() bool { return true }

// ToCaddyConfig 生成 Caddy 配置片段
func (d *PublishDomain) ToCaddyConfig() string {
    return fmt.Sprintf(`%s {
    root * %s
    file_server
}`, d.Domain, d.TargetPath)
}
```

**Caddy 集成说明**:
- Caddy 自动申请和续签 Let's Encrypt 证书
- 只需将域名配置添加到 Caddyfile 即可
- `ToCaddyConfig()` 方法生成对应的 Caddy 配置片段

---

### 任务 5: Sync Domain - 业务逻辑 ⭐ 优先级: 中

**文件**: `internal/domain/sync/entity/manager.go`

```go
package entity

import (
    "fmt"
    "time"
    
    adminVO "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

type Manager struct {
    config      *adminVO.CouchDBConfig
    couchClient CouchDBClient
    repo        Repository
}

type CouchDBClient interface {
    CreateDatabase(name string) error
    CreateUser(email, password string) error
    GetDatabaseInfo(name string) (*DatabaseInfo, error)
}

type DatabaseInfo struct {
    DocCount int   `json:"doc_count"`
    DiskSize int64 `json:"disk_size"`
}

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

func NewManager(config *adminVO.CouchDBConfig, client CouchDBClient, repo Repository) *Manager {
    return &Manager{
        config:      config,
        couchClient: client,
        repo:        repo,
    }
}

// ========== 设备/IP 验证 (治理逻辑) ==========

// ValidateAndRecordAccess 验证设备和 IP，记录访问
func (m *Manager) ValidateAndRecordAccess(licenseKey, deviceID, deviceName, ipAddress string) error {
    license, err := m.repo.GetLicenseByKey(licenseKey)
    if err != nil {
        return fmt.Errorf("license not found: %w", err)
    }
    
    if !license.IsValid() {
        return fmt.Errorf("license is not valid")
    }
    
    // 检查设备
    if err := m.checkAndRecordDevice(license, deviceID, deviceName); err != nil {
        return err
    }
    
    // 检查 IP
    if err := m.checkAndRecordIP(license, ipAddress); err != nil {
        return err
    }
    
    return nil
}

func (m *Manager) checkAndRecordDevice(license *contentVO.License, deviceID, deviceName string) error {
    now := time.Now().UnixMilli()
    
    // 查找已存在的设备
    existingDevice, err := m.repo.GetDeviceByID(license.LicenseKey, deviceID)
    if err == nil && existingDevice != nil {
        // 更新访问记录
        existingDevice.LastSeenAt = now
        existingDevice.AccessCount++
        return m.repo.SaveDevice(existingDevice)
    }
    
    // 新设备 - 检查限制
    if !license.CanAddDevice() {
        return fmt.Errorf("device limit reached (%d/%d)", license.CurrentDevices, license.MaxDevices)
    }
    
    // 添加新设备
    device := &contentVO.LicenseDevice{
        License:     license.QueryString(),
        DeviceID:    deviceID,
        DeviceName:  deviceName,
        FirstSeenAt: now,
        LastSeenAt:  now,
        AccessCount: 1,
        Status:      "active",
    }
    
    if err := m.repo.SaveDevice(device); err != nil {
        return err
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
        // 更新访问记录
        existingIP.LastSeenAt = now
        existingIP.AccessCount++
        return m.repo.SaveIP(existingIP)
    }
    
    // 新 IP - 检查限制
    if !license.CanAddIP() {
        return fmt.Errorf("IP limit reached (%d/%d)", license.CurrentIPs, license.MaxIPs)
    }
    
    // 添加新 IP
    ip := &contentVO.LicenseIP{
        License:     license.QueryString(),
        IPAddress:   ipAddress,
        FirstSeenAt: now,
        LastSeenAt:  now,
        AccessCount: 1,
        Status:      "active",
    }
    
    if err := m.repo.SaveIP(ip); err != nil {
        return err
    }
    
    // 更新 License IP 计数
    license.CurrentIPs++
    return m.repo.UpdateLicense(license)
}

// ========== CouchDB 账号分配 ==========

func (m *Manager) CreateSyncAccount(license *contentVO.License) (*contentVO.SyncAccount, error) {
    // 检查是否已存在
    existing, _ := m.repo.GetSyncAccountByLicense(license.LicenseKey)
    if existing != nil {
        return existing, nil
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
    
    account := &contentVO.SyncAccount{
        License:    license.QueryString(),
        Email:      email,
        DBName:     dbName,
        DBEndpoint: fmt.Sprintf("%s/%s", m.config.URL, dbName),
        Status:     "active",
        CreatedAt:  time.Now().UnixMilli(),
    }
    
    if err := m.repo.SaveSyncAccount(account); err != nil {
        return nil, err
    }
    
    return account, nil
}

// ========== 使用量监控 ==========

func (m *Manager) UpdateUsage(licenseKey string) (*contentVO.SyncUsage, error) {
    account, err := m.repo.GetSyncAccountByLicense(licenseKey)
    if err != nil {
        return nil, err
    }
    
    dbInfo, err := m.couchClient.GetDatabaseInfo(account.DBName)
    if err != nil {
        return nil, err
    }
    
    usage := &contentVO.SyncUsage{
        SyncAccount:   account.QueryString(),
        DocumentCount: dbInfo.DocCount,
        StorageBytes:  dbInfo.DiskSize,
        LastSyncAt:    time.Now().UnixMilli(),
        RecordedAt:    time.Now().UnixMilli(),
    }
    
    if err := m.repo.SaveSyncUsage(usage); err != nil {
        return nil, err
    }
    
    return usage, nil
}
```

---

### 任务 6: Publish Domain - 业务逻辑 ⭐ 优先级: 中

**文件**: `internal/domain/publish/entity/manager.go`

```go
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

type Manager struct {
    fs   afero.Fs
    repo Repository
}

type Repository interface {
    SavePublishSite(site *contentVO.PublishSite) error
    GetPublishSitesByLicense(licenseKey string) ([]contentVO.PublishSite, error)
    SavePublishUsage(usage *contentVO.PublishUsage) error
    
    // 自定义域名
    SavePublishDomain(domain *contentVO.PublishDomain) error
    GetPublishDomainBySite(siteID string) (*contentVO.PublishDomain, error)
    GetAllActivePublishDomains() ([]contentVO.PublishDomain, error)
}

func NewManager(repo Repository) *Manager {
    return &Manager{
        fs:   afero.NewOsFs(),
        repo: repo,
    }
}

// ========== 站点部署 ==========

func (m *Manager) DeploySite(license *contentVO.License, site *contentVO.PublishSite) error {
    userDir := license.ToUserDir()
    
    var targetDir string
    if site.SiteType == "article" {
        targetDir = filepath.Join(application.PublishDir(), userDir, "articles", site.Name)
    } else {
        targetDir = filepath.Join(application.PublishDir(), userDir, "sites", site.Name)
    }
    
    if err := application.EnsureDirExists(targetDir); err != nil {
        return err
    }
    
    absAssetPath, err := site.AbsAssetPath(application.UploadDir())
    if err != nil {
        return err
    }
    
    if err := zip.Unzip(absAssetPath, targetDir); err != nil {
        return err
    }
    
    site.FolderPath = targetDir
    site.PublicURL = fmt.Sprintf("/%s/%s/%ss/%s", 
        application.PublishFolder(), userDir, site.SiteType, site.Name)
    site.Status = "active"
    
    return m.repo.SavePublishSite(site)
}

// ========== 容量管理 ==========

func (m *Manager) UpdateUsage(licenseKey string) (*contentVO.PublishUsage, error) {
    sites, err := m.repo.GetPublishSitesByLicense(licenseKey)
    if err != nil {
        return nil, err
    }
    
    var totalSize int64
    for _, site := range sites {
        size, _ := m.getDirSize(site.FolderPath)
        totalSize += size
    }
    
    usage := &contentVO.PublishUsage{
        License:      licenseKey,
        SiteCount:    len(sites),
        StorageBytes: totalSize,
        RecordedAt:   time.Now().UnixMilli(),
    }
    
    if err := m.repo.SavePublishUsage(usage); err != nil {
        return nil, err
    }
    
    return usage, nil
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

func (m *Manager) BindCustomDomain(license *contentVO.License, site *contentVO.PublishSite, domain string) (*contentVO.PublishDomain, error) {
    // 检查权限
    features := license.GetFeatures()
    if !features.CustomDomain {
        return nil, fmt.Errorf("custom domain not enabled for this plan")
    }
    
    publishDomain := &contentVO.PublishDomain{
        License:     license.QueryString(),
        PublishSite: site.QueryString(),
        Domain:      domain,
        TargetPath:  site.FolderPath,
        Status:      "active",
        CreatedAt:   time.Now().UnixMilli(),
        UpdatedAt:   time.Now().UnixMilli(),
    }
    
    if err := m.repo.SavePublishDomain(publishDomain); err != nil {
        return nil, err
    }
    
    // 更新 Caddy 配置
    if err := m.updateCaddyConfig(); err != nil {
        return nil, fmt.Errorf("failed to update caddy config: %w", err)
    }
    
    return publishDomain, nil
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
        config += d.ToCaddyConfig() + "\n\n"
    }
    
    // 写入 Caddyfile
    caddyfilePath := filepath.Join(application.DataDir(), "Caddyfile")
    if err := afero.WriteFile(m.fs, caddyfilePath, []byte(config), 0644); err != nil {
        return err
    }
    
    // 重载 Caddy 配置 (caddy reload)
    // 可以通过 Caddy Admin API 或命令行实现
    return nil
}
```

---

## 4. 目录结构

```
~/.local/share/hugoverse/
├── preview/             # 现有 - 临时预览
├── publish/             # 新增 - 用户发布
│   ├── {userDir1}/      # hash(license.ToEmail())[:16]
│   │   ├── sites/
│   │   │   ├── my-blog/
│   │   │   └── portfolio/
│   │   └── articles/
│   │       ├── article-1/
│   │       └── article-2/
│   └── {userDir2}/
└── ...
```

---

## 5. 权限矩阵

| 套餐 | 设备数 | IP数 | Sync | Sync配额 | Publish | 站点数 | 存储 | 自定义域名 | 有效期 |
|-----|-------|-----|------|---------|---------|-------|------|----------|-------|
| Free | 1 | 1 | ❌ | - | ❌ | 0 | 100MB | ❌ | 7天 |
| Starter | 3 | 3 | ✅ | 500MB | ✅ | 3 | 1GB | ❌ | 1年 |
| Creator | 5 | 5 | ✅ | 2GB | ✅ | 10 | 5GB | ✅ | 1年 |
| Pro | 10 | 10 | ✅ | 10GB | ✅ | 50 | 20GB | ✅ | 1年 |
| Enterprise | 无限制 | 无限制 | ✅ | 50GB | ✅ | 无限制 | 100GB | ✅ | 1年 |

---

## 6. 实现优先级

### Phase 1: 基础结构 (2天)
- [ ] 任务 1: License ValueObject (含设备/IP 限制)
- [ ] 任务 2: LicenseDevice/LicenseIP 数据表
- [ ] Application 目录函数
- [ ] 注册为内容类型

### Phase 2: Sync 功能 (3天)
- [ ] 任务 3: SyncAccount/SyncUsage 数据表
- [ ] 任务 5: Sync Manager (设备/IP 验证 + CouchDB 分配)
- [ ] Admin CouchDB 配置

### Phase 3: Publish 功能 (3天)
- [ ] 任务 4: PublishSite/PublishUsage/PublishDomain 数据表
- [ ] 任务 6: Publish Manager (部署 + 容量 + 域名)
- [ ] Publish API Handler

---

## 7. 关键代码参考

| 功能 | 参考文件 |
|-----|---------|
| 目录管理 | `internal/application/dir.go` → `PreviewDir()` |
| 资源部署 | `internal/interfaces/api/handler/handlemdf.go` |
| 文件上传 | `internal/domain/content/valueobject/previewmdf.go` |
| Zip 解压 | `pkg/zip/zip.go` |
| 用户数据库 | `pkg/db/cache.go` |
