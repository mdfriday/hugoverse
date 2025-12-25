# License API 集成实现方案

> 创建时间: 2024-12-25
> 目的: 实现 CouchDB Client、License API Handler 并集成到现有 API Server

---

## 1. 需求概述

基于已完成的 License ValueObject 和 Manager 实现，需要：

1. **CouchDB Client** - 与 CouchDB 服务器交互的真实实现
2. **License API Handler** - 处理 License 激活、设备验证、Sync/Publish 服务
3. **API Server 集成** - 将 Handler 注册到现有服务器

---

## 2. 架构设计

### 2.1 模块关系

```
internal/
├── infrastructure/
│   └── couchdb/
│       └── client.go           # CouchDB HTTP Client
├── domain/
│   ├── sync/entity/manager.go  # 依赖 CouchDBClient 接口
│   └── publish/entity/manager.go
└── interfaces/api/
    ├── handler/
    │   └── handlelicense.go    # License API Handler
    ├── handlers.go             # 注册新的 License 路由
    └── server.go               # 初始化 Sync/Publish Manager
```

### 2.2 依赖注入流程

```
Server 启动
    ↓
创建 CouchDB Client (infrastructure)
    ↓
创建 Sync Manager (domain) ← 注入 CouchDB Client
    ↓
创建 Publish Manager (domain)
    ↓
创建 License Handler (interface) ← 注入 Managers
    ↓
注册 API 路由
```

---

## 3. 任务拆解

### 任务 1: CouchDB Client 实现 ⭐ 优先级: 高

**文件**: `internal/infrastructure/couchdb/client.go`

```go
package couchdb

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    adminVO "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
    syncEntity "github.com/mdfriday/hugoverse/internal/domain/sync/entity"
)

// Client CouchDB HTTP 客户端
type Client struct {
    config     *adminVO.CouchDBConfig
    httpClient *http.Client
}

// NewClient 创建 CouchDB 客户端
func NewClient(config *adminVO.CouchDBConfig) *Client {
    return &Client{
        config: config,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// 实现 syncEntity.CouchDBClient 接口

// CreateDatabase 创建数据库
func (c *Client) CreateDatabase(name string) error {
    url := fmt.Sprintf("%s/%s", c.config.URL, name)
    req, err := http.NewRequest(http.MethodPut, url, nil)
    if err != nil {
        return err
    }
    c.setBasicAuth(req)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 201 Created 或 412 Already exists 都算成功
    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusPreconditionFailed {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to create database: %s", string(body))
    }
    
    return nil
}

// CreateUser 创建 CouchDB 用户
func (c *Client) CreateUser(email, password string) error {
    userDoc := map[string]interface{}{
        "_id":      fmt.Sprintf("org.couchdb.user:%s", email),
        "name":     email,
        "password": password,
        "roles":    []string{},
        "type":     "user",
    }
    
    body, _ := json.Marshal(userDoc)
    url := fmt.Sprintf("%s/_users/org.couchdb.user:%s", c.config.URL, email)
    
    req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    c.setBasicAuth(req)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 201 Created 或 409 Conflict (已存在) 都算成功
    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to create user: %s", string(respBody))
    }
    
    return nil
}

// SetDatabasePermission 设置数据库权限
func (c *Client) SetDatabasePermission(dbName, email string) error {
    securityDoc := map[string]interface{}{
        "admins": map[string]interface{}{
            "names": []string{},
            "roles": []string{"_admin"},
        },
        "members": map[string]interface{}{
            "names": []string{email},
            "roles": []string{},
        },
    }
    
    body, _ := json.Marshal(securityDoc)
    url := fmt.Sprintf("%s/%s/_security", c.config.URL, dbName)
    
    req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    c.setBasicAuth(req)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to set permission: %s", string(respBody))
    }
    
    return nil
}

// GetDatabaseInfo 获取数据库信息
func (c *Client) GetDatabaseInfo(name string) (*syncEntity.DatabaseInfo, error) {
    url := fmt.Sprintf("%s/%s", c.config.URL, name)
    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }
    c.setBasicAuth(req)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("database not found: %s", name)
    }
    
    var info struct {
        DocCount int   `json:"doc_count"`
        DiskSize int64 `json:"disk_size"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
        return nil, err
    }
    
    return &syncEntity.DatabaseInfo{
        DocCount: info.DocCount,
        DiskSize: info.DiskSize,
    }, nil
}

func (c *Client) setBasicAuth(req *http.Request) {
    req.SetBasicAuth(c.config.AdminUser, c.config.AdminPass)
}

// 确保实现接口
var _ syncEntity.CouchDBClient = (*Client)(nil)
```

---

### 任务 2: License Repository 实现 ⭐ 优先级: 高

**文件**: `internal/infrastructure/repository/license_repo.go`

```go
package repository

import (
    "encoding/json"
    "fmt"
    
    "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
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

func NewLicenseRepository(db *database.Database) *LicenseRepository {
    return &LicenseRepository{db: db}
}

// ========== License 操作 ==========

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

func (r *LicenseRepository) UpdateLicense(license *valueobject.License) error {
    data, err := json.Marshal(license)
    if err != nil {
        return err
    }
    return r.db.PutContent(license, data)
}

func (r *LicenseRepository) CreateLicense(license *valueobject.License) error {
    license.SetHash()
    data, err := json.Marshal(license)
    if err != nil {
        return err
    }
    return r.db.NewContent(license, data)
}

// ========== Device 操作 ==========

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

func (r *LicenseRepository) SaveDevice(device *valueobject.LicenseDevice) error {
    device.SetHash()
    data, err := json.Marshal(device)
    if err != nil {
        return err
    }
    
    if device.ID == -1 || device.ID == 0 {
        return r.db.NewContent(device, data)
    }
    return r.db.PutContent(device, data)
}

// ========== IP 操作 ==========

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

func (r *LicenseRepository) SaveIP(ip *valueobject.LicenseIP) error {
    ip.SetHash()
    data, err := json.Marshal(ip)
    if err != nil {
        return err
    }
    
    if ip.ID == -1 || ip.ID == 0 {
        return r.db.NewContent(ip, data)
    }
    return r.db.PutContent(ip, data)
}

// ========== SyncAccount 操作 ==========

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

func (r *LicenseRepository) SaveSyncAccount(account *valueobject.SyncAccount) error {
    account.SetHash()
    data, err := json.Marshal(account)
    if err != nil {
        return err
    }
    
    if account.ID == -1 || account.ID == 0 {
        return r.db.NewContent(account, data)
    }
    return r.db.PutContent(account, data)
}

func (r *LicenseRepository) SaveSyncUsage(usage *valueobject.SyncUsage) error {
    usage.SetHash()
    data, err := json.Marshal(usage)
    if err != nil {
        return err
    }
    return r.db.NewContent(usage, data)
}
```

---

### 任务 3: License API Handler ⭐ 优先级: 高

**文件**: `internal/interfaces/api/handler/handlelicense.go`

```go
package handler

import (
    "encoding/json"
    "net/http"
    "time"
    
    contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
    syncEntity "github.com/mdfriday/hugoverse/internal/domain/sync/entity"
    publishEntity "github.com/mdfriday/hugoverse/internal/domain/publish/entity"
)

// LicenseHandler License 相关的 API Handler
type LicenseHandler struct {
    syncManager    *syncEntity.Manager
    publishManager *publishEntity.Manager
    repo           LicenseRepository
}

// LicenseRepository Handler 需要的 Repository 接口
type LicenseRepository interface {
    GetLicenseByKey(key string) (*contentVO.License, error)
    UpdateLicense(license *contentVO.License) error
    CreateLicense(license *contentVO.License) error
}

// NewLicenseHandler 创建 License Handler
func NewLicenseHandler(
    syncManager *syncEntity.Manager,
    publishManager *publishEntity.Manager,
    repo LicenseRepository,
) *LicenseHandler {
    return &LicenseHandler{
        syncManager:    syncManager,
        publishManager: publishManager,
        repo:           repo,
    }
}

// ========== API Handlers ==========

// ActivateHandler 激活 License
// POST /api/license/v2/activate
func (h *LicenseHandler) ActivateHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        LicenseKey string `json:"license_key"`
        DeviceID   string `json:"device_id"`
        DeviceName string `json:"device_name"`
        DeviceType string `json:"device_type"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.jsonError(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // 获取客户端 IP
    ipAddress := h.getClientIP(r)
    
    // 获取或创建 License
    license, err := h.repo.GetLicenseByKey(req.LicenseKey)
    if err != nil {
        // License 不存在，创建新的
        license = &contentVO.License{
            LicenseKey:     req.LicenseKey,
            Plan:           contentVO.PlanStarter, // 默认 Starter
            Activated:      true,
            ActivatedAt:    time.Now().UnixMilli(),
            ExpiryDate:     time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
            MaxDevices:     3,
            MaxIPs:         3,
            CurrentDevices: 0,
            CurrentIPs:     0,
        }
        if err := h.repo.CreateLicense(license); err != nil {
            h.jsonError(w, "Failed to create license", http.StatusInternalServerError)
            return
        }
    }
    
    // 验证设备和 IP
    if err := h.syncManager.ValidateAndRecordAccess(
        req.LicenseKey, req.DeviceID, req.DeviceName, req.DeviceType, ipAddress,
    ); err != nil {
        h.jsonError(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // 创建 Sync 账号 (如果支持)
    var syncAccount *contentVO.SyncAccount
    if license.GetFeatures().SyncEnabled {
        syncAccount, _ = h.syncManager.CreateSyncAccount(license)
    }
    
    // 返回响应
    response := map[string]interface{}{
        "success":     true,
        "license_key": license.LicenseKey,
        "plan":        license.Plan,
        "features":    license.GetFeatures(),
        "expires_at":  license.ExpiryDate,
    }
    
    if syncAccount != nil {
        response["sync"] = map[string]interface{}{
            "email":       syncAccount.Email,
            "db_endpoint": syncAccount.DBEndpoint,
        }
    }
    
    h.jsonResponse(w, response)
}

// GetLicenseHandler 获取 License 信息
// GET /api/license/v2/info?key=xxx
func (h *LicenseHandler) GetLicenseHandler(w http.ResponseWriter, r *http.Request) {
    licenseKey := r.URL.Query().Get("key")
    if licenseKey == "" {
        h.jsonError(w, "License key is required", http.StatusBadRequest)
        return
    }
    
    license, err := h.repo.GetLicenseByKey(licenseKey)
    if err != nil {
        h.jsonError(w, "License not found", http.StatusNotFound)
        return
    }
    
    response := map[string]interface{}{
        "license_key":     license.LicenseKey,
        "plan":            license.Plan,
        "activated":       license.Activated,
        "expires_at":      license.ExpiryDate,
        "is_expired":      license.IsExpired(),
        "is_valid":        license.IsValid(),
        "current_devices": license.CurrentDevices,
        "max_devices":     license.MaxDevices,
        "current_ips":     license.CurrentIPs,
        "max_ips":         license.MaxIPs,
        "features":        license.GetFeatures(),
    }
    
    h.jsonResponse(w, response)
}

// GetDevicesHandler 获取 License 的设备列表
// GET /api/license/v2/devices?key=xxx
func (h *LicenseHandler) GetDevicesHandler(w http.ResponseWriter, r *http.Request) {
    licenseKey := r.URL.Query().Get("key")
    if licenseKey == "" {
        h.jsonError(w, "License key is required", http.StatusBadRequest)
        return
    }
    
    devices, err := h.syncManager.GetDevices(licenseKey)
    if err != nil {
        h.jsonError(w, "Failed to get devices", http.StatusInternalServerError)
        return
    }
    
    h.jsonResponse(w, map[string]interface{}{
        "devices": devices,
        "count":   len(devices),
    })
}

// GetSyncInfoHandler 获取 Sync 信息
// GET /api/license/v2/sync?key=xxx
func (h *LicenseHandler) GetSyncInfoHandler(w http.ResponseWriter, r *http.Request) {
    licenseKey := r.URL.Query().Get("key")
    if licenseKey == "" {
        h.jsonError(w, "License key is required", http.StatusBadRequest)
        return
    }
    
    account, err := h.syncManager.GetSyncAccount(licenseKey)
    if err != nil {
        h.jsonError(w, "Sync account not found", http.StatusNotFound)
        return
    }
    
    // 获取使用量
    usage, _ := h.syncManager.UpdateUsage(licenseKey)
    
    response := map[string]interface{}{
        "email":       account.Email,
        "db_name":     account.DBName,
        "db_endpoint": account.DBEndpoint,
        "status":      account.Status,
    }
    
    if usage != nil {
        response["usage"] = map[string]interface{}{
            "document_count": usage.DocumentCount,
            "storage_bytes":  usage.StorageBytes,
            "quota_bytes":    usage.QuotaBytes,
            "percentage":     usage.UsagePercentage(),
        }
    }
    
    h.jsonResponse(w, response)
}

// ========== Helper Methods ==========

func (h *LicenseHandler) getClientIP(r *http.Request) string {
    // 优先检查 X-Forwarded-For
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return xff
    }
    // 检查 X-Real-IP
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    // 使用 RemoteAddr
    return r.RemoteAddr
}

func (h *LicenseHandler) jsonResponse(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func (h *LicenseHandler) jsonError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": false,
        "error":   message,
    })
}
```

---

### 任务 4: API Server 集成 ⭐ 优先级: 高

**修改文件**: `internal/interfaces/api/handlers.go`

```go
// 在 registerLicenseHandler 函数中添加新的路由

func (s *Server) registerLicenseHandler() {
    // 现有的 License Handler (保留)
    licenseHandler, err := NewLicenseHandler()
    if err != nil {
        fmt.Printf("Warning: Failed to initialize license handler: %v\n", err)
    } else {
        s.mux.HandleFunc("/api/license/activate", s.wrapLicensePostHandler(licenseHandler.ActivateLicenseHandler))
        s.mux.HandleFunc("/api/license/public-keys", s.wrapPublicHandler(licenseHandler.GetPublicKeysHandler))
        s.mux.HandleFunc("/api/license/validate", s.wrapLicensePostHandler(licenseHandler.ValidateLicenseKeyHandler))
        s.mux.HandleFunc("/api/license/decrypt", s.wrapLicensePostHandler(licenseHandler.DecryptContentHandler))
    }
    
    // 新的 License V2 API (基于新的 Manager)
    if s.handler.licenseHandler != nil {
        // 激活和信息
        s.mux.HandleFunc("/api/license/v2/activate", s.wrapLicensePostHandler(s.handler.licenseHandler.ActivateHandler))
        s.mux.HandleFunc("/api/license/v2/info", s.wrapPublicHandler(s.handler.licenseHandler.GetLicenseHandler))
        
        // 设备管理
        s.mux.HandleFunc("/api/license/v2/devices", s.wrapPublicHandler(s.handler.licenseHandler.GetDevicesHandler))
        
        // Sync 服务
        s.mux.HandleFunc("/api/license/v2/sync", s.wrapPublicHandler(s.handler.licenseHandler.GetSyncInfoHandler))
    }
}
```

---

## 4. API 接口文档

### 4.1 License V2 API

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/license/v2/activate` | 激活 License 并记录设备/IP |
| GET | `/api/license/v2/info?key=xxx` | 获取 License 详情 |
| GET | `/api/license/v2/devices?key=xxx` | 获取设备列表 |
| GET | `/api/license/v2/sync?key=xxx` | 获取 Sync 账号信息 |

### 4.2 请求/响应示例

**激活 License**:
```bash
curl -X POST http://localhost:1314/api/license/v2/activate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "MDF-ABCD-EFGH-JKLM",
    "device_id": "device-uuid-1234",
    "device_name": "MacBook Pro",
    "device_type": "desktop"
  }'
```

响应:
```json
{
  "success": true,
  "license_key": "MDF-ABCD-EFGH-JKLM",
  "plan": "starter",
  "features": {
    "max_devices": 3,
    "sync_enabled": true,
    "custom_domain": false
  },
  "sync": {
    "email": "abcd-efgh-jklm@mdfriday.com",
    "db_endpoint": "http://localhost:5984/userdb-abc123"
  }
}
```

---

## 5. 实现顺序 ✅ 已完成

### Phase 1: 基础设施层 ✅
1. [x] 创建 `internal/infrastructure/couchdb/client.go` ✅
2. [x] 创建 `internal/infrastructure/repository/license_repo.go` ✅

### Phase 2: Handler 层 ✅
3. [x] 创建 `internal/interfaces/api/handler/handlelicense.go` ✅
4. [x] 创建 `internal/infrastructure/couchdb/client_test.go` ✅
5. [x] 创建 `internal/interfaces/api/handler/handlelicense_test.go` ✅

### Phase 3: 集成 ✅
6. [x] 修改 `internal/interfaces/api/handlers.go` 注册路由 ✅

---

## 6. 测试验证

```bash
# 1. 启动服务器
go run main.go serve

# 2. 测试激活 API
curl -X POST http://localhost:1314/api/license/v2/activate \
  -H "Content-Type: application/json" \
  -d '{"license_key":"MDF-TEST-1234-5678","device_id":"test-device","device_name":"Test","device_type":"desktop"}'

# 3. 查询 License 信息
curl "http://localhost:1314/api/license/v2/info?key=MDF-TEST-1234-5678"

# 4. 查询设备列表
curl "http://localhost:1314/api/license/v2/devices?key=MDF-TEST-1234-5678"
```

