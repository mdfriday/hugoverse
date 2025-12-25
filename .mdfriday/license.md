# License 管理系统需求与实现计划

## 1. 需求概述

实现 license 管理系统，可以支持用户创建 sync 服务和 publish 服务。

用户可以在前端激活 license ，就可以知道 license 的类型和有效期。
license 有 Free，Starter，Creator，Pro, Enterprise 不同的类型，有效期通常都是一年。

用户的使用流程是，在前端输入 license , 会发送一个 activate 的请求到后端，验证是否合法，有没有过期， 如果是合法有效的，则会在后端创建一个 couchDB 的数据库。
用户就可以用这个账号进行文件同步了。
同时，我们也会为用户生成一个专属的 WEB SERVER 文件夹，里面可以放用户分享的单篇文章，或者是整个站点，而且每一个都是以独立文件夹存在的，所以我们还可以绑定自定义域名。

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
internal/domain/
├── content/valueobject/
│   ├── license.go           # License 数据结构 (内容资源)
│   ├── syncaccount.go       # Sync 账号关系表
│   ├── syncusage.go         # Sync 使用量记录
│   ├── publishsite.go       # Publish 站点记录
│   └── publishusage.go      # Publish 容量记录
├── admin/valueobject/
│   ├── couchdb.go           # CouchDB 配置 (仅配置，无业务逻辑)
│   └── publishconfig.go     # Publish 服务配置
├── sync/                    # 新 Domain - Sync 业务
│   ├── entity/
│   │   └── manager.go       # Sync 管理器
│   ├── factory/
│   └── type.go
└── publish/                 # 新 Domain - Publish 业务
    ├── entity/
    │   └── manager.go       # Publish 管理器
    ├── factory/
    └── type.go
```

### 2.3 各领域职责

| 领域 | 职责 |
|-----|------|
| `content/valueobject/license.go` | License 数据结构，作为内容资源存储 |
| `content/valueobject/sync*.go` | Sync 相关的关系数据表 |
| `content/valueobject/publish*.go` | Publish 相关的关系数据表 |
| `admin/valueobject/couchdb.go` | CouchDB 连接配置 (仅配置) |
| `admin/valueobject/publishconfig.go` | Publish 服务配置 (仅配置) |
| `sync/` (新 Domain) | Sync 业务逻辑：分配 CouchDB 账号、监控使用量 |
| `publish/` (新 Domain) | Publish 业务逻辑：管理发布内容和容量，参考现有 preview 实现 |

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

// LicensePlan 定义 License 类型
type LicensePlan string

const (
    PlanFree       LicensePlan = "free"       // 免费版 - 7天试用
    PlanStarter    LicensePlan = "starter"    // 入门版 - 1年
    PlanCreator    LicensePlan = "creator"    // 创作者版 - 1年
    PlanPro        LicensePlan = "pro"        // 专业版 - 1年
    PlanEnterprise LicensePlan = "enterprise" // 企业版 - 1年
)

// License 作为 Content ValueObject
type License struct {
    Item // 嵌入基础 Item 结构

    // 基本信息
    LicenseKey string      `json:"license_key"`  // MDF-XXXX-XXXX-XXXX
    Plan       LicensePlan `json:"plan"`         // 套餐类型
    
    // 有效期
    IssueDate  int64 `json:"issue_date"`  // 签发时间戳
    ExpiryDate int64 `json:"expiry_date"` // 过期时间戳
    
    // 激活状态
    Activated  bool  `json:"activated"`   // 是否已激活
    ActivatedAt int64 `json:"activated_at"` // 激活时间
}

// MarshalEditor 实现 editor.Editable 接口
func (l *License) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(l,
        editor.Field{
            View: editor.Input("LicenseKey", l, map[string]string{
                "label":       "License Key",
                "type":        "text",
                "placeholder": "MDF-XXXX-XXXX-XXXX",
            }),
        },
        editor.Field{
            View: editor.Select("Plan", l, map[string]string{
                "label": "Plan",
            }, map[string]string{
                "free":       "Free (7 days)",
                "starter":    "Starter (1 year)",
                "creator":    "Creator (1 year)",
                "pro":        "Pro (1 year)",
                "enterprise": "Enterprise (1 year)",
            }),
        },
    )
    return view, err
}

// String 返回显示名称
func (l *License) String() string {
    return fmt.Sprintf("%s (%s)", l.LicenseKey, l.Plan)
}

func (l *License) SetHash() {
    l.Hash = hash.MD5(l.LicenseKey)
}

func (l *License) IndexContent() bool {
    return true
}

// ========== License 转用户机制 ==========

// ToEmail 将 License Key 转换为邮箱
// MDF-ABCD-EFGH-JKLM -> abcd-efgh-jklm@mdfriday.com
func (l *License) ToEmail() string {
    // 移除 "MDF-" 前缀，转小写
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return fmt.Sprintf("%s@mdfriday.com", key)
}

// ToPassword 将 License Key 转换为密码
// MDF-ABCD-EFGH-JKLM -> base64("abcd-efgh-jklm")
func (l *License) ToPassword() string {
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return base64.StdEncoding.EncodeToString([]byte(key))
}

// ToUserDir 生成用户目录名
func (l *License) ToUserDir() string {
    return hash.MD5(l.ToEmail())[:16]
}

// IsExpired 检查是否过期
func (l *License) IsExpired() bool {
    return time.Now().UnixMilli() > l.ExpiryDate
}

// IsValid 检查是否有效
func (l *License) IsValid() bool {
    return l.Activated && !l.IsExpired()
}

// GetFeatures 获取功能权限
func (l *License) GetFeatures() *LicenseFeatures {
    return GetPlanFeatures(l.Plan)
}
```

**权限定义** (同文件):
```go
// LicenseFeatures 定义各套餐功能权限
type LicenseFeatures struct {
    MaxSites    int   `json:"max_sites"`    // 最大站点数
    MaxStorageMB int  `json:"max_storage"`  // 存储空间 (MB)
    SyncEnabled bool  `json:"sync_enabled"` // 同步功能
    SyncQuotaMB int   `json:"sync_quota"`   // 同步配额 (MB)
    PublishEnabled bool `json:"publish_enabled"` // 发布功能
    CustomDomain bool `json:"custom_domain"` // 自定义域名
}

func GetPlanFeatures(plan LicensePlan) *LicenseFeatures {
    switch plan {
    case PlanFree:
        return &LicenseFeatures{
            MaxSites: 1, MaxStorageMB: 100, SyncEnabled: false, 
            SyncQuotaMB: 0, PublishEnabled: false, CustomDomain: false,
        }
    case PlanStarter:
        return &LicenseFeatures{
            MaxSites: 3, MaxStorageMB: 1024, SyncEnabled: true, 
            SyncQuotaMB: 500, PublishEnabled: true, CustomDomain: false,
        }
    case PlanCreator:
        return &LicenseFeatures{
            MaxSites: 10, MaxStorageMB: 5120, SyncEnabled: true, 
            SyncQuotaMB: 2048, PublishEnabled: true, CustomDomain: true,
        }
    case PlanPro:
        return &LicenseFeatures{
            MaxSites: 50, MaxStorageMB: 20480, SyncEnabled: true, 
            SyncQuotaMB: 10240, PublishEnabled: true, CustomDomain: true,
        }
    case PlanEnterprise:
        return &LicenseFeatures{
            MaxSites: -1, MaxStorageMB: 102400, SyncEnabled: true, 
            SyncQuotaMB: 51200, PublishEnabled: true, CustomDomain: true,
        }
    default:
        return &LicenseFeatures{}
    }
}
```

---

### 任务 2: Content Domain - Sync 关系数据表 ⭐ 优先级: 高

#### 2.1 SyncAccount - CouchDB 账号记录

**文件**: `internal/domain/content/valueobject/syncaccount.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// SyncAccount License 对应的 CouchDB 账号
type SyncAccount struct {
    Item

    License    string `json:"license"`     // 关联的 License (QueryString)
    Email      string `json:"email"`       // CouchDB 用户邮箱 (xxx@mdfriday.com)
    DBName     string `json:"db_name"`     // 分配的数据库名
    DBEndpoint string `json:"db_endpoint"` // 数据库访问端点
    
    Status     string `json:"status"`      // active / suspended / deleted
    CreatedAt  int64  `json:"created_at"`
}

func (s *SyncAccount) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(s,
        editor.Field{
            View: editor.RefSelect("License", s, map[string]string{
                "label": "License",
            }, "License", `{{ .license_key }}`, s.refSelData["License"]),
        },
        editor.Field{
            View: editor.Input("Email", s, map[string]string{
                "label": "User Email",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Input("DBName", s, map[string]string{
                "label": "Database Name",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Input("Status", s, map[string]string{
                "label": "Status",
                "type":  "text",
            }),
        },
    )
    return view, err
}

func (s *SyncAccount) String() string {
    return fmt.Sprintf("%s - %s", s.Email, s.DBName)
}

func (s *SyncAccount) SelectContentTypes() []string {
    return []string{"License"}
}

func (s *SyncAccount) IndexContent() bool {
    return true
}

// 持有 refSelData 用于关联选择
var refSelData map[string][][]byte

func (s *SyncAccount) SetSelectData(data map[string][][]byte) {
    s.refSelData = data
}

// 临时字段，不序列化
type syncAccountInternal struct {
    refSelData map[string][][]byte
}

// 为 SyncAccount 添加内部字段
func (s *SyncAccount) initInternal() {
    // 初始化逻辑
}
```

#### 2.2 SyncUsage - 使用量记录

**文件**: `internal/domain/content/valueobject/syncusage.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// SyncUsage Sync 使用量记录
type SyncUsage struct {
    Item

    SyncAccount string `json:"sync_account"` // 关联的 SyncAccount
    
    // 使用量统计
    DocumentCount int   `json:"document_count"` // 文档数量
    StorageBytes  int64 `json:"storage_bytes"`  // 存储大小 (字节)
    LastSyncAt    int64 `json:"last_sync_at"`   // 最后同步时间
    
    // 配额
    QuotaBytes    int64 `json:"quota_bytes"`    // 配额上限 (字节)
    UsagePercent  int   `json:"usage_percent"`  // 使用百分比
    
    RecordedAt    int64 `json:"recorded_at"`    // 记录时间
}

func (s *SyncUsage) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(s,
        editor.Field{
            View: editor.RefSelect("SyncAccount", s, map[string]string{
                "label": "Sync Account",
            }, "SyncAccount", `{{ .email }}`, nil),
        },
        editor.Field{
            View: editor.Input("DocumentCount", s, map[string]string{
                "label": "Document Count",
                "type":  "number",
            }),
        },
        editor.Field{
            View: editor.Input("StorageBytes", s, map[string]string{
                "label": "Storage (bytes)",
                "type":  "number",
            }),
        },
    )
    return view, err
}

func (s *SyncUsage) String() string {
    return fmt.Sprintf("%s - %d docs, %d bytes", s.SyncAccount, s.DocumentCount, s.StorageBytes)
}

func (s *SyncUsage) IndexContent() bool {
    return true
}
```

---

### 任务 3: Content Domain - Publish 关系数据表 ⭐ 优先级: 高

#### 3.1 PublishSite - 发布站点记录

**文件**: `internal/domain/content/valueobject/publishsite.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// PublishSite 发布站点记录
type PublishSite struct {
    Item

    License     string `json:"license"`      // 关联的 License
    SiteName    string `json:"site_name"`    // 站点名称
    SiteType    string `json:"site_type"`    // site / article
    
    // 路径信息
    FolderPath  string `json:"folder_path"`  // 站点文件夹路径
    PublicURL   string `json:"public_url"`   // 公开访问 URL
    
    // 域名
    CustomDomain string `json:"custom_domain"` // 自定义域名 (可选)
    DomainStatus string `json:"domain_status"` // pending / active / failed
    
    // 状态
    Status     string `json:"status"`       // active / deleted
    CreatedAt  int64  `json:"created_at"`
    UpdatedAt  int64  `json:"updated_at"`
}

func (p *PublishSite) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(p,
        editor.Field{
            View: editor.RefSelect("License", p, map[string]string{
                "label": "License",
            }, "License", `{{ .license_key }}`, nil),
        },
        editor.Field{
            View: editor.Input("SiteName", p, map[string]string{
                "label": "Site Name",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Select("SiteType", p, map[string]string{
                "label": "Site Type",
            }, map[string]string{
                "site":    "Full Site",
                "article": "Single Article",
            }),
        },
        editor.Field{
            View: editor.Input("CustomDomain", p, map[string]string{
                "label":       "Custom Domain",
                "type":        "text",
                "placeholder": "blog.example.com",
            }),
        },
    )
    return view, err
}

func (p *PublishSite) String() string {
    return fmt.Sprintf("%s (%s)", p.SiteName, p.SiteType)
}

func (p *PublishSite) IndexContent() bool {
    return true
}
```

#### 3.2 PublishUsage - 容量记录

**文件**: `internal/domain/content/valueobject/publishusage.go`

```go
package valueobject

import (
    "fmt"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// PublishUsage Publish 容量记录
type PublishUsage struct {
    Item

    License      string `json:"license"`       // 关联的 License
    
    // 使用量
    SiteCount    int    `json:"site_count"`    // 站点数量
    StorageBytes int64  `json:"storage_bytes"` // 总存储大小
    
    // 配额
    MaxSites     int    `json:"max_sites"`     // 最大站点数
    QuotaBytes   int64  `json:"quota_bytes"`   // 存储配额
    
    RecordedAt   int64  `json:"recorded_at"`
}

func (p *PublishUsage) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(p,
        editor.Field{
            View: editor.RefSelect("License", p, map[string]string{
                "label": "License",
            }, "License", `{{ .license_key }}`, nil),
        },
        editor.Field{
            View: editor.Input("SiteCount", p, map[string]string{
                "label": "Site Count",
                "type":  "number",
            }),
        },
        editor.Field{
            View: editor.Input("StorageBytes", p, map[string]string{
                "label": "Storage (bytes)",
                "type":  "number",
            }),
        },
    )
    return view, err
}

func (p *PublishUsage) String() string {
    return fmt.Sprintf("%d sites, %d bytes", p.SiteCount, p.StorageBytes)
}
```

---

### 任务 4: Admin Domain - 配置信息 ⭐ 优先级: 中

#### 4.1 CouchDB 配置

**文件**: `internal/domain/admin/valueobject/couchdb.go`

```go
package valueobject

import (
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// CouchDBConfig CouchDB 服务器配置 (仅配置信息)
type CouchDBConfig struct {
    contentVO.Item

    URL      string `json:"url"`       // http://localhost:5984
    Username string `json:"username"`  // admin
    Password string `json:"password"`  // 密码
    DBPrefix string `json:"db_prefix"` // mdf_sync_
}

func (c *CouchDBConfig) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(c,
        editor.Field{
            View: editor.Input("URL", c, map[string]string{
                "label":       "CouchDB URL",
                "placeholder": "http://localhost:5984",
            }),
        },
        editor.Field{
            View: editor.Input("Username", c, map[string]string{
                "label": "Admin Username",
            }),
        },
        editor.Field{
            View: editor.Input("Password", c, map[string]string{
                "label": "Admin Password",
                "type":  "password",
            }),
        },
        editor.Field{
            View: editor.Input("DBPrefix", c, map[string]string{
                "label":       "Database Prefix",
                "placeholder": "mdf_sync_",
            }),
        },
    )
    return view, err
}
```

#### 4.2 Publish 配置

**文件**: `internal/domain/admin/valueobject/publishconfig.go`

```go
package valueobject

import (
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/pkg/editor"
)

// PublishConfig Publish 服务配置 (仅配置信息)
type PublishConfig struct {
    contentVO.Item

    BaseDir   string `json:"base_dir"`   // 发布根目录 /var/www/mdfriday
    BaseURL   string `json:"base_url"`   // 公开访问基础 URL
    MaxSizeMB int    `json:"max_size"`   // 单站点最大容量
}

func (p *PublishConfig) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(p,
        editor.Field{
            View: editor.Input("BaseDir", p, map[string]string{
                "label":       "Publish Base Directory",
                "placeholder": "/var/www/mdfriday/publish",
            }),
        },
        editor.Field{
            View: editor.Input("BaseURL", p, map[string]string{
                "label":       "Public Base URL",
                "placeholder": "https://publish.mdfriday.com",
            }),
        },
        editor.Field{
            View: editor.Input("MaxSizeMB", p, map[string]string{
                "label": "Max Site Size (MB)",
                "type":  "number",
            }),
        },
    )
    return view, err
}
```

---

### 任务 5: Sync Domain ⭐ 优先级: 中

**目标**: 实现 Sync 业务逻辑

**目录结构**:
```
internal/domain/sync/
├── entity/
│   └── manager.go
├── factory/
│   └── sync.go
└── type.go
```

**文件**: `internal/domain/sync/type.go`

```go
package sync

import (
    "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// Manager Sync 管理器接口
type Manager interface {
    // 为 License 创建同步账号
    CreateAccount(license *valueobject.License) (*valueobject.SyncAccount, error)
    
    // 获取同步账号
    GetAccount(licenseKey string) (*valueobject.SyncAccount, error)
    
    // 更新使用量
    UpdateUsage(accountID string) (*valueobject.SyncUsage, error)
    
    // 检查配额
    CheckQuota(accountID string) (bool, error)
    
    // 暂停账号 (超出配额)
    SuspendAccount(accountID string) error
}

// CouchDBClient CouchDB 操作接口
type CouchDBClient interface {
    CreateDatabase(name string) error
    DeleteDatabase(name string) error
    GetDatabaseInfo(name string) (*DatabaseInfo, error)
    CreateUser(email, password string) error
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
    DocCount   int   `json:"doc_count"`
    DiskSize   int64 `json:"disk_size"`
    UpdateSeq  string `json:"update_seq"`
}
```

**文件**: `internal/domain/sync/entity/manager.go`

```go
package entity

import (
    "fmt"
    "time"
    
    "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/internal/domain/sync"
)

// Manager Sync 管理器实现
type Manager struct {
    config      *valueobject.CouchDBConfig
    couchClient sync.CouchDBClient
    repo        ContentRepository
}

// ContentRepository 内容仓库接口
type ContentRepository interface {
    SaveSyncAccount(account *contentVO.SyncAccount) error
    GetSyncAccountByLicense(licenseKey string) (*contentVO.SyncAccount, error)
    SaveSyncUsage(usage *contentVO.SyncUsage) error
}

func NewManager(config *valueobject.CouchDBConfig, client sync.CouchDBClient, repo ContentRepository) *Manager {
    return &Manager{
        config:      config,
        couchClient: client,
        repo:        repo,
    }
}

// CreateAccount 为 License 创建同步账号
func (m *Manager) CreateAccount(license *contentVO.License) (*contentVO.SyncAccount, error) {
    // 检查是否已存在
    existing, _ := m.repo.GetSyncAccountByLicense(license.LicenseKey)
    if existing != nil {
        return existing, nil
    }
    
    // 生成账号信息
    email := license.ToEmail()
    password := license.ToPassword()
    dbName := fmt.Sprintf("%s%s", m.config.DBPrefix, license.ToUserDir())
    
    // 在 CouchDB 创建数据库
    if err := m.couchClient.CreateDatabase(dbName); err != nil {
        return nil, fmt.Errorf("failed to create database: %w", err)
    }
    
    // 创建用户
    if err := m.couchClient.CreateUser(email, password); err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    // 保存账号记录
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

// UpdateUsage 更新使用量
func (m *Manager) UpdateUsage(accountID string) (*contentVO.SyncUsage, error) {
    account, err := m.repo.GetSyncAccountByLicense(accountID)
    if err != nil {
        return nil, err
    }
    
    // 获取数据库信息
    dbInfo, err := m.couchClient.GetDatabaseInfo(account.DBName)
    if err != nil {
        return nil, err
    }
    
    // 创建使用量记录
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

### 任务 6: Publish Domain ⭐ 优先级: 中

**目标**: 实现 Publish 业务逻辑 (参考现有 preview 实现)

**参考代码**:
- `internal/domain/content/entity/preview.go` - Preview 处理逻辑
- `internal/domain/content/entity/hugo.go` - previewDir() 方法
- `internal/interfaces/api/handler/handlepreview.go` - Preview API

**目录结构**:
```
internal/domain/publish/
├── entity/
│   └── manager.go
├── factory/
│   └── publish.go
└── type.go
```

**文件**: `internal/domain/publish/type.go`

```go
package publish

import (
    "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// Manager Publish 管理器接口
type Manager interface {
    // 初始化用户发布目录
    InitUserDir(license *valueobject.License) (string, error)
    
    // 创建站点
    CreateSite(license *valueobject.License, siteName, siteType string) (*valueobject.PublishSite, error)
    
    // 部署站点内容
    DeploySite(siteID string, files map[string][]byte) error
    
    // 获取用户所有站点
    GetUserSites(licenseKey string) ([]valueobject.PublishSite, error)
    
    // 绑定自定义域名
    BindDomain(siteID, domain string) error
    
    // 更新容量统计
    UpdateUsage(licenseKey string) (*valueobject.PublishUsage, error)
    
    // 检查配额
    CheckQuota(licenseKey string) (bool, error)
}
```

**文件**: `internal/domain/publish/entity/manager.go`

```go
package entity

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    
    "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/pkg/hash"
    "github.com/spf13/afero"
)

// Manager Publish 管理器实现
type Manager struct {
    config *valueobject.PublishConfig
    fs     afero.Fs
    repo   ContentRepository
}

type ContentRepository interface {
    SavePublishSite(site *contentVO.PublishSite) error
    GetPublishSitesByLicense(licenseKey string) ([]contentVO.PublishSite, error)
    SavePublishUsage(usage *contentVO.PublishUsage) error
}

func NewManager(config *valueobject.PublishConfig, repo ContentRepository) *Manager {
    return &Manager{
        config: config,
        fs:     afero.NewOsFs(),
        repo:   repo,
    }
}

// InitUserDir 初始化用户发布目录
// 类似现有的 previewDir() 实现
func (m *Manager) InitUserDir(license *contentVO.License) (string, error) {
    userDir := license.ToUserDir()
    userPath := filepath.Join(m.config.BaseDir, userDir)
    
    // 创建目录结构
    dirs := []string{
        userPath,
        filepath.Join(userPath, "sites"),
        filepath.Join(userPath, "articles"),
    }
    
    for _, dir := range dirs {
        if err := m.fs.MkdirAll(dir, 0755); err != nil {
            return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }
    
    return userPath, nil
}

// CreateSite 创建站点
func (m *Manager) CreateSite(license *contentVO.License, siteName, siteType string) (*contentVO.PublishSite, error) {
    userDir := license.ToUserDir()
    
    var sitePath string
    if siteType == "article" {
        sitePath = filepath.Join(m.config.BaseDir, userDir, "articles", siteName)
    } else {
        sitePath = filepath.Join(m.config.BaseDir, userDir, "sites", siteName)
    }
    
    // 创建站点目录
    if err := m.fs.MkdirAll(sitePath, 0755); err != nil {
        return nil, err
    }
    
    // 生成公开 URL
    publicURL := fmt.Sprintf("%s/%s/%s/%s", m.config.BaseURL, userDir, siteType+"s", siteName)
    
    site := &contentVO.PublishSite{
        License:    license.QueryString(),
        SiteName:   siteName,
        SiteType:   siteType,
        FolderPath: sitePath,
        PublicURL:  publicURL,
        Status:     "active",
        CreatedAt:  time.Now().UnixMilli(),
    }
    
    if err := m.repo.SavePublishSite(site); err != nil {
        return nil, err
    }
    
    return site, nil
}

// DeploySite 部署站点内容 (参考 static.Copy 实现)
func (m *Manager) DeploySite(siteID string, files map[string][]byte) error {
    // 获取站点信息
    // 写入文件到站点目录
    for filename, content := range files {
        filePath := filepath.Join("", filename) // 需要获取站点路径
        
        if err := m.fs.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
            return err
        }
        
        if err := afero.WriteFile(m.fs, filePath, content, 0644); err != nil {
            return err
        }
    }
    
    return nil
}

// UpdateUsage 更新容量统计
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
```

---

## 4. 用户目录结构

参考现有的 preview 实现 (`~/.local/share/hugoverse/preview/`)，为每个 License 用户创建类似的结构：

```
{PublishBaseDir}/
├── {userDir1}/                          # hash(license.ToEmail())[:16]
│   ├── sites/
│   │   ├── my-blog/                     # 站点 1
│   │   │   ├── index.html
│   │   │   └── ...
│   │   └── portfolio/                   # 站点 2
│   └── articles/
│       ├── article-1/                   # 单篇文章 1
│       └── article-2/
├── {userDir2}/
│   └── ...
```

---

## 5. 权限矩阵

| 套餐 | 最大站点 | 存储空间 | Sync | Sync 配额 | Publish | 自定义域名 | 有效期 |
|-----|---------|---------|------|----------|---------|----------|-------|
| Free | 1 | 100MB | ❌ | - | ❌ | ❌ | 7天 |
| Starter | 3 | 1GB | ✅ | 500MB | ✅ | ❌ | 1年 |
| Creator | 10 | 5GB | ✅ | 2GB | ✅ | ✅ | 1年 |
| Pro | 50 | 20GB | ✅ | 10GB | ✅ | ✅ | 1年 |
| Enterprise | 无限制 | 100GB | ✅ | 50GB | ✅ | ✅ | 1年 |

---

## 6. 实现优先级与里程碑

### Phase 1: 数据结构 (2天)
- [ ] 任务 1: License ValueObject
- [ ] 任务 2: Sync 关系数据表
- [ ] 任务 3: Publish 关系数据表
- [ ] 注册为内容类型

### Phase 2: 配置管理 (1天)
- [ ] 任务 4: Admin 配置 ValueObject
- [ ] 在后台添加配置页面

### Phase 3: Sync Domain (3天)
- [ ] 任务 5: Sync Manager 实现
- [ ] CouchDB 客户端
- [ ] 使用量监控

### Phase 4: Publish Domain (3天)
- [ ] 任务 6: Publish Manager 实现
- [ ] 参考 preview 实现
- [ ] 容量管理

### Phase 5: API 集成 (2天)
- [ ] License 激活 API
- [ ] Sync/Publish 状态 API

---

## 7. 关键代码参考

| 参考内容 | 文件路径 |
|---------|---------|
| ValueObject 示例 | `internal/domain/content/valueobject/preview.go` |
| Item 基类 | `internal/domain/content/valueobject/item.go` |
| Preview 目录创建 | `internal/domain/content/entity/hugo.go` → `previewDir()` |
| Preview API | `internal/interfaces/api/handler/handlepreview.go` |
| 文件同步 | `pkg/fs/static/static.go` |
| 用户数据库 | `pkg/db/cache.go` → `OpenUserStore()` |
