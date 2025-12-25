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
internal/
├── application/
│   └── dir.go                   # 添加 PublishDir() 函数 (参考 PreviewDir)
├── domain/
│   ├── content/valueobject/
│   │   ├── license.go           # License 数据结构 (内容资源)
│   │   ├── syncaccount.go       # Sync 账号关系表
│   │   ├── syncusage.go         # Sync 使用量记录
│   │   ├── publishsite.go       # Publish 站点记录
│   │   └── publishusage.go      # Publish 容量记录
│   ├── admin/valueobject/
│   │   └── couchdb.go           # CouchDB 配置 (仅配置，无业务逻辑)
│   ├── sync/                    # 新 Domain - Sync 业务
│   │   └── entity/manager.go
│   └── publish/                 # 新 Domain - Publish 业务
│       └── entity/manager.go    # 参考现有 preview/deploy 实现
└── interfaces/api/handler/
    ├── handlepublish.go         # 参考 handlepreview.go, handlemdf.go
    └── ...
```

### 2.3 参考现有实现

**目录结构 (参考 `application/dir.go`)**:
```go
// 现有实现
func PreviewDir() string {
    return filepath.Join(DataDir(), "preview")  // ~/.local/share/hugoverse/preview
}

// 新增 - Publish 目录
func PublishDir() string {
    return filepath.Join(DataDir(), "publish")  // ~/.local/share/hugoverse/publish
}
```

**API 处理 (参考 `handlemdf.go`)**:
```go
// DeployMDFridayPreviewHandler 现有逻辑:
// 1. 获取 MDFPreview 对象
// 2. 解压 zip 到 preview/{name} 目录
// 3. 返回 /preview/{name} 路径

// 新的 Publish 逻辑:
// 1. 验证 License
// 2. 获取 userDir (license.ToUserDir())
// 3. 解压到 publish/{userDir}/sites/{name} 或 publish/{userDir}/articles/{name}
// 4. 返回 /publish/{userDir}/sites/{name} 路径
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
    Item

    LicenseKey string      `json:"license_key"`  // MDF-XXXX-XXXX-XXXX
    Plan       LicensePlan `json:"plan"`
    
    IssueDate   int64 `json:"issue_date"`
    ExpiryDate  int64 `json:"expiry_date"`
    
    Activated   bool  `json:"activated"`
    ActivatedAt int64 `json:"activated_at"`
}

// MarshalEditor 实现 editor.Editable 接口
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

// ToEmail: MDF-ABCD-EFGH-JKLM -> abcd-efgh-jklm@mdfriday.com
func (l *License) ToEmail() string {
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return fmt.Sprintf("%s@mdfriday.com", key)
}

// ToPassword: base64("abcd-efgh-jklm")
func (l *License) ToPassword() string {
    key := strings.ToLower(strings.TrimPrefix(l.LicenseKey, "MDF-"))
    return base64.StdEncoding.EncodeToString([]byte(key))
}

// ToUserDir: hash(email)[:16]
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
```

**权限定义**:
```go
type LicenseFeatures struct {
    MaxSites     int  `json:"max_sites"`
    MaxStorageMB int  `json:"max_storage"`
    SyncEnabled  bool `json:"sync_enabled"`
    SyncQuotaMB  int  `json:"sync_quota"`
    PublishEnabled bool `json:"publish_enabled"`
    CustomDomain bool `json:"custom_domain"`
}

func GetPlanFeatures(plan LicensePlan) *LicenseFeatures {
    switch plan {
    case PlanFree:
        return &LicenseFeatures{1, 100, false, 0, false, false}
    case PlanStarter:
        return &LicenseFeatures{3, 1024, true, 500, true, false}
    case PlanCreator:
        return &LicenseFeatures{10, 5120, true, 2048, true, true}
    case PlanPro:
        return &LicenseFeatures{50, 20480, true, 10240, true, true}
    case PlanEnterprise:
        return &LicenseFeatures{-1, 102400, true, 51200, true, true}
    default:
        return &LicenseFeatures{}
    }
}
```

---

### 任务 2: Application - 目录函数 ⭐ 优先级: 高

**文件**: `internal/application/dir.go` (修改)

```go
// 新增常量
const folderPublish = "publish"

// 新增函数 (参考 PreviewDir)
func PublishDir() string {
    return filepath.Join(DataDir(), folderPublish)
}

func PublishFolder() string {
    return folderPublish
}

// 在 init() 中添加
func init() {
    // ... 现有代码 ...
    
    err = EnsureDirExists(PublishDir())
    if err != nil {
        log.Fatalln(err)
    }
}

// 更新 dir 结构体
func (d *dir) PublishDir() string {
    return PublishDir()
}
func (d *dir) PublishFolder() string {
    return folderPublish
}
```

---

### 任务 3: Content Domain - Sync 关系数据表 ⭐ 优先级: 中

**文件**: `internal/domain/content/valueobject/syncaccount.go`

```go
package valueobject

// SyncAccount License 对应的 CouchDB 账号
type SyncAccount struct {
    Item

    License    string `json:"license"`     // 关联的 License
    Email      string `json:"email"`       // CouchDB 用户邮箱
    DBName     string `json:"db_name"`     // 分配的数据库名
    DBEndpoint string `json:"db_endpoint"` // 数据库访问端点
    Status     string `json:"status"`      // active / suspended
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

// PublishSite 发布站点记录 (参考 MDFPreview 结构)
type PublishSite struct {
    Item

    License      string `json:"license"`       // 关联的 License
    Name         string `json:"name"`          // 站点名称
    SiteType     string `json:"site_type"`     // site / article
    Asset        string `json:"asset"`         // 上传的 zip 文件
    Size         string `json:"size"`          // 文件大小
    FolderPath   string `json:"folder_path"`   // 部署后的路径
    PublicURL    string `json:"public_url"`    // 公开访问 URL
    CustomDomain string `json:"custom_domain"` // 自定义域名
    Status       string `json:"status"`        // pending / active / deleted
    CreatedAt    int64  `json:"created_at"`
}

func (p *PublishSite) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(p,
        editor.Field{
            View: editor.Input("Name", p, map[string]string{
                "label": "Site Name",
                "type":  "text",
            }),
        },
        editor.Field{
            View: editor.Select("SiteType", p, map[string]string{
                "label": "Type",
            }, map[string]string{
                "site": "Full Site", "article": "Article",
            }),
        },
        editor.Field{
            View: editor.File("Asset", p, map[string]string{
                "label": "Asset (zip)",
            }),
        },
    )
    return view, err
}

func (p *PublishSite) String() string { return p.Name }
func (p *PublishSite) Deploy() bool   { return true }
func (p *PublishSite) IndexContent() bool { return true }

// AbsAssetPath 获取资源绝对路径 (参考 MDFPreview)
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

---

### 任务 5: Admin Domain - CouchDB 配置 ⭐ 优先级: 中

**文件**: `internal/domain/admin/valueobject/couchdb.go`

```go
package valueobject

// CouchDBConfig CouchDB 服务器配置 (仅配置信息，无业务逻辑)
type CouchDBConfig struct {
    contentVO.Item

    URL      string `json:"url"`       // http://localhost:5984
    Username string `json:"username"`
    Password string `json:"password"`
    DBPrefix string `json:"db_prefix"` // mdf_sync_
}

func (c *CouchDBConfig) MarshalEditor() ([]byte, error) {
    view, err := editor.Form(c,
        editor.Field{
            View: editor.Input("URL", c, map[string]string{
                "label": "CouchDB URL",
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
                "label": "Database Prefix",
            }),
        },
    )
    return view, err
}
```

---

### 任务 6: Publish Domain - 业务逻辑 ⭐ 优先级: 中

**参考现有实现**:
- `handlemdf.go` → `DeployMDFridayPreviewHandler` 
- `handlepreview.go` → `PreviewContentHandler`

**文件**: `internal/domain/publish/entity/manager.go`

```go
package entity

import (
    "fmt"
    "path/filepath"
    
    "github.com/mdfriday/hugoverse/internal/application"
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/pkg/zip"
    "github.com/spf13/afero"
)

type Manager struct {
    fs   afero.Fs
    repo ContentRepository
}

type ContentRepository interface {
    SavePublishSite(site *contentVO.PublishSite) error
    GetPublishSitesByLicense(licenseKey string) ([]contentVO.PublishSite, error)
}

func NewManager(repo ContentRepository) *Manager {
    return &Manager{
        fs:   afero.NewOsFs(),
        repo: repo,
    }
}

// DeploySite 部署站点 (参考 DeployMDFridayPreviewHandler)
func (m *Manager) DeploySite(license *contentVO.License, site *contentVO.PublishSite) error {
    // 1. 获取用户目录
    userDir := license.ToUserDir()
    
    // 2. 构建目标路径: ~/.local/share/hugoverse/publish/{userDir}/sites/{name}
    var targetDir string
    if site.SiteType == "article" {
        targetDir = filepath.Join(application.PublishDir(), userDir, "articles", site.Name)
    } else {
        targetDir = filepath.Join(application.PublishDir(), userDir, "sites", site.Name)
    }
    
    // 3. 确保目录存在
    if err := application.EnsureDirExists(targetDir); err != nil {
        return err
    }
    
    // 4. 解压资源 (参考 handlemdf.go)
    absAssetPath, err := site.AbsAssetPath(application.UploadDir())
    if err != nil {
        return err
    }
    
    if err := zip.Unzip(absAssetPath, targetDir); err != nil {
        return err
    }
    
    // 5. 更新站点信息
    site.FolderPath = targetDir
    site.PublicURL = fmt.Sprintf("/%s/%s/%ss/%s", 
        application.PublishFolder(), userDir, site.SiteType, site.Name)
    site.Status = "active"
    
    return m.repo.SavePublishSite(site)
}

// GetUserDir 获取用户发布目录
func (m *Manager) GetUserDir(license *contentVO.License) string {
    return filepath.Join(application.PublishDir(), license.ToUserDir())
}
```

---

### 任务 7: API Handler ⭐ 优先级: 中

**文件**: `internal/interfaces/api/handler/handlepublish.go` (新建，参考 handlemdf.go)

```go
package handler

import (
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/mdfriday/hugoverse/internal/application"
    "github.com/mdfriday/hugoverse/internal/domain/content"
    "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    "github.com/mdfriday/hugoverse/pkg/loggers"
    "github.com/mdfriday/hugoverse/pkg/zip"
    "path/filepath"
)

// DeployPublishSiteHandler 部署用户站点
// 参考 DeployMDFridayPreviewHandler 实现
func (s *Handler) DeployPublishSiteHandler(res http.ResponseWriter, req *http.Request) {
    q := req.URL.Query()
    id := q.Get("id")
    t := q.Get("type")
    licenseKey := req.FormValue("license_key")

    loggers.SetGlobalFields(s.newLogFields("deploy publish site"))

    // 1. 验证 License
    license, err := s.contentApp.GetLicenseByKey(licenseKey)
    if err != nil || !license.IsValid() {
        s.log.Errorf("Invalid license: %v", err)
        res.WriteHeader(http.StatusUnauthorized)
        return
    }

    // 2. 检查权限
    features := license.GetFeatures()
    if !features.PublishEnabled {
        s.log.Errorf("Publish not enabled for this plan")
        res.WriteHeader(http.StatusForbidden)
        return
    }

    // 3. 获取 PublishSite 对象
    pt, ok := s.contentApp.GetContentCreator(t)
    if !ok {
        res.WriteHeader(http.StatusNotFound)
        return
    }

    p := pt()
    _, ok = p.(content.Deployable)
    if !ok {
        res.WriteHeader(http.StatusBadRequest)
        return
    }

    sc, err := s.contentApp.GetContentObject(t, id)
    if err != nil {
        s.log.Errorf("Error getting content: %v", err)
        res.WriteHeader(http.StatusInternalServerError)
        return
    }

    site, ok := sc.(*valueobject.PublishSite)
    if !ok {
        res.WriteHeader(http.StatusInternalServerError)
        return
    }

    // 4. 获取用户目录
    userDir := license.ToUserDir()
    
    // 5. 构建目标路径
    var targetDir string
    if site.SiteType == "article" {
        targetDir = filepath.Join(application.PublishDir(), userDir, "articles", site.Name)
    } else {
        targetDir = filepath.Join(application.PublishDir(), userDir, "sites", site.Name)
    }

    // 6. 确保目录存在
    if err := application.EnsureDirExists(targetDir); err != nil {
        s.log.Errorf("Error creating directory: %v", err)
        res.WriteHeader(http.StatusInternalServerError)
        return
    }

    // 7. 解压资源
    absAssetPath, err := site.AbsAssetPath(application.UploadDir())
    if err != nil {
        s.log.Errorf("Error getting asset path: %v", err)
        res.WriteHeader(http.StatusInternalServerError)
        return
    }

    if err := zip.Unzip(absAssetPath, targetDir); err != nil {
        s.log.Errorf("Error unzipping: %v", err)
        res.WriteHeader(http.StatusInternalServerError)
        return
    }

    // 8. 返回公开 URL
    publicURL := fmt.Sprintf("/%s/%s/%ss/%s", 
        application.PublishFolder(), userDir, site.SiteType, site.Name)
    
    jsonBytes, _ := json.Marshal(publicURL)
    j, _ := s.res.FmtJSON(jsonBytes)

    res.WriteHeader(http.StatusOK)
    s.res.Json(res, j)
}
```

---

## 4. 目录结构

**现有 Preview 目录**:
```
~/.local/share/hugoverse/
├── preview/
│   ├── {shortLink1}/    # 随机短链接
│   └── {shortLink2}/
└── ...
```

**新增 Publish 目录** (参考 Preview):
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

| 套餐 | 最大站点 | 存储空间 | Sync | Sync 配额 | Publish | 自定义域名 | 有效期 |
|-----|---------|---------|------|----------|---------|----------|-------|
| Free | 1 | 100MB | ❌ | - | ❌ | ❌ | 7天 |
| Starter | 3 | 1GB | ✅ | 500MB | ✅ | ❌ | 1年 |
| Creator | 10 | 5GB | ✅ | 2GB | ✅ | ✅ | 1年 |
| Pro | 50 | 20GB | ✅ | 10GB | ✅ | ✅ | 1年 |
| Enterprise | 无限制 | 100GB | ✅ | 50GB | ✅ | ✅ | 1年 |

---

## 6. 实现优先级

### Phase 1: 基础结构 (2天)
- [ ] 任务 1: License ValueObject
- [ ] 任务 2: Application 目录函数
- [ ] 注册为内容类型

### Phase 2: Publish 功能 (2天)
- [ ] 任务 4: PublishSite/PublishUsage ValueObject
- [ ] 任务 6: Publish Manager
- [ ] 任务 7: Publish API Handler

### Phase 3: Sync 功能 (3天)
- [ ] 任务 3: SyncAccount/SyncUsage ValueObject
- [ ] 任务 5: CouchDB 配置
- [ ] Sync Manager 实现

---

## 7. 关键代码参考

| 功能 | 参考文件 |
|-----|---------|
| 目录管理 | `internal/application/dir.go` → `PreviewDir()` |
| 资源部署 | `internal/interfaces/api/handler/handlemdf.go` → `DeployMDFridayPreviewHandler` |
| 文件上传 | `internal/domain/content/valueobject/previewmdf.go` → `MDFPreview` |
| Zip 解压 | `pkg/zip/zip.go` → `Unzip()` |
| 用户数据库 | `pkg/db/cache.go` → `OpenUserStore()` |
