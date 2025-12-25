# License 管理系统验证操作手册

> 创建时间: 2024-12-25
> 目的: 验证 License 管理系统的实现是否正确

---

## 1. 编译验证

### 1.1 编译整个项目

```shell
cd /Users/sunwei/github/mdfriday/hugoverse

# 编译所有 internal 包
go build ./internal/...

# 如果成功，无输出
echo "编译成功: $?"
```

### 1.2 运行单元测试

```shell
# 运行所有新增的测试
go test -v ./internal/domain/content/valueobject/... \
  ./internal/domain/sync/... \
  ./internal/domain/publish/...

# 只运行 License 相关测试
go test -v ./internal/domain/content/valueobject/... -run License

# 运行 Sync Manager 测试
go test -v ./internal/domain/sync/entity/...

# 运行 Publish Manager 测试
go test -v ./internal/domain/publish/entity/...
```

---

## 2. 代码结构验证

### 2.1 检查新创建的文件

```shell
# ValueObjects
ls -la internal/domain/content/valueobject/license*.go
ls -la internal/domain/content/valueobject/sync*.go
ls -la internal/domain/content/valueobject/publish*.go

# Managers
ls -la internal/domain/sync/entity/
ls -la internal/domain/publish/entity/

# Admin Config
ls -la internal/domain/admin/valueobject/couchdb.go
```

### 2.2 检查类型注册

```shell
# 查看 factory/content.go 中的注册
grep -A 20 "prepareAdminTypes" internal/domain/content/factory/content.go
```

预期输出应包含：
- `License`
- `LicenseDevice`
- `LicenseIP`
- `SyncAccount`
- `SyncUsage`
- `PublishSite`
- `PublishUsage`
- `PublishDomain`

---

## 3. 功能点验证

### 3.1 License ValueObject 验证

```go
// 在 Go 代码中测试 (可以写成测试用例或临时 main)
package main

import (
    "fmt"
    vo "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

func main() {
    license := &vo.License{
        LicenseKey: "MDF-ABCD-EFGH-JKLM",
        Plan:       vo.PlanStarter,
        Activated:  true,
        ExpiryDate: 9999999999999,
        MaxDevices: 3,
        MaxIPs:     3,
    }
    
    // 1. 验证转用户机制
    fmt.Println("Email:", license.ToEmail())
    // 预期: abcd-efgh-jklm@mdfriday.com
    
    fmt.Println("Password:", license.ToPassword())
    // 预期: YWJjZC1lZmdoLWprbG0=
    
    fmt.Println("UserDir:", license.ToUserDir())
    // 预期: 16位 MD5 前缀
    
    // 2. 验证 Hash 设置
    license.SetHash()
    fmt.Println("Hash:", license.Hash)
    // 预期: 32位 MD5
    
    // 3. 验证功能特性
    features := license.GetFeatures()
    fmt.Printf("Features: MaxDevices=%d, SyncEnabled=%v, CustomDomain=%v\n",
        features.MaxDevices, features.SyncEnabled, features.CustomDomain)
    // Starter: MaxDevices=3, SyncEnabled=true, CustomDomain=false
}
```

### 3.2 权限矩阵验证

| 套餐 | 设备数 | IP数 | Sync | 自定义域名 |
|-----|-------|-----|------|----------|
| Free | 1 | 1 | ❌ | ❌ |
| Starter | 3 | 3 | ✅ | ❌ |
| Creator | 5 | 5 | ✅ | ✅ |
| Pro | 10 | 10 | ✅ | ✅ |
| Enterprise | -1 | -1 | ✅ | ✅ |

```shell
# 运行权限测试
go test -v ./internal/domain/content/valueobject/... -run GetPlanFeatures
```

---

## 4. BoltDB 存储验证

### 4.1 Bucket 命名规则检查

新增的 Bucket 应该包括：

| Bucket 名称 | 说明 |
|------------|------|
| `license` | License 主存储 |
| `license__index` | License slug 索引 |
| `licensedevice` | 设备记录 |
| `licensedevice__index` | 设备 slug 索引 |
| `licenseip` | IP 记录 |
| `licenseip__index` | IP slug 索引 |
| `syncaccount` | Sync 账号 |
| `syncusage` | Sync 使用量 |
| `publishsite` | 发布站点 |
| `publishusage` | Publish 容量 |
| `publishdomain` | 自定义域名 |

### 4.2 Hash 索引验证

```go
// 验证 Hash 生成规则
license.SetHash()  // MD5(LicenseKey)
device.SetHash()   // MD5(License + ":" + DeviceID)
ip.SetHash()       // MD5(License + ":" + IPAddress)
```

---

## 5. Sync Manager 验证

### 5.1 设备/IP 限制测试

```shell
go test -v ./internal/domain/sync/entity/... -run TestManagerDeviceLimit
go test -v ./internal/domain/sync/entity/... -run TestManagerValidateAndRecordAccess
```

### 5.2 CouchDB 账号创建测试

```shell
go test -v ./internal/domain/sync/entity/... -run TestManagerCreateSyncAccount
```

验证点：
- ✅ Starter/Creator/Pro/Enterprise 可以创建 Sync 账号
- ✅ Free 计划无法创建 Sync 账号
- ✅ 数据库名称格式: `userdb-{userDir}`
- ✅ 用户邮箱格式: `xxx-xxx-xxx@mdfriday.com`

---

## 6. Publish Manager 验证

### 6.1 自定义域名测试

```shell
go test -v ./internal/domain/publish/entity/... -run TestPublishManagerBindCustomDomain
go test -v ./internal/domain/publish/entity/... -run TestPublishManagerBindCustomDomainFreePlan
```

验证点：
- ✅ Creator/Pro/Enterprise 可以绑定自定义域名
- ✅ Free/Starter 无法绑定自定义域名

### 6.2 Caddy 配置生成测试

```shell
go test -v ./internal/domain/publish/entity/... -run TestPublishManagerGenerateCaddyConfig
```

预期 Caddy 配置格式：
```
blog.example.com {
    root * /data/publish/user123/sites/my-blog
    file_server
}
```

---

## 7. 目录结构验证

### 7.1 检查 PublishDir 函数

```shell
grep -A 5 "PublishDir" internal/application/dir.go
grep -A 5 "PublishFolder" internal/application/dir.go
```

### 7.2 预期目录结构

```
~/.local/share/hugoverse/
├── preview/           # 现有 - 临时预览
├── publish/           # 新增 - 用户发布
│   └── {userDir}/     # hash(license.ToEmail())[:16]
│       ├── sites/
│       │   └── my-blog/
│       └── articles/
│           └── article-1/
└── Caddyfile          # 自动生成的 Caddy 配置
```

---

## 8. 一键测试命令

```shell
# 完整测试脚本
cd /Users/sunwei/github/mdfriday/hugoverse

echo "=== 1. 编译验证 ==="
go build ./internal/... && echo "✅ 编译成功" || echo "❌ 编译失败"

echo ""
echo "=== 2. 运行所有 License 相关测试 ==="
go test ./internal/domain/content/valueobject/... \
  ./internal/domain/sync/entity/... \
  ./internal/domain/publish/entity/... \
  -count=1

echo ""
echo "=== 3. 测试覆盖率 ==="
go test ./internal/domain/content/valueobject/... \
  ./internal/domain/sync/entity/... \
  ./internal/domain/publish/entity/... \
  -cover

echo ""
echo "=== 4. 验证新文件存在 ==="
for f in license licensedevice licenseip syncaccount syncusage publishsite publishusage publishdomain; do
  if [ -f "internal/domain/content/valueobject/${f}.go" ]; then
    echo "✅ ${f}.go 存在"
  else
    echo "❌ ${f}.go 不存在"
  fi
done

echo ""
echo "=== 验证完成 ==="
```

---

## 9. 问题排查

### 9.1 常见问题

| 问题 | 解决方案 |
|-----|---------|
| 编译失败 | 检查 import 路径是否正确 |
| 测试失败 | 查看具体测试输出，检查 Mock 实现 |
| Bucket 未创建 | 确认类型已在 `factory/content.go` 中注册 |

### 9.2 查看详细测试输出

```shell
# 显示详细输出
go test -v ./internal/domain/sync/entity/... 2>&1 | tee test-output.log

# 只显示失败的测试
go test ./... 2>&1 | grep -E "(FAIL|---)"
```

---

## 10. 后续集成步骤

- [x] 实现 CouchDB Client 真实接口 ✅
- [x] 添加 License API Handler ✅
- [x] 集成到现有 API Server ✅
- [ ] 实现 Caddy 配置热重载
- [ ] 添加 E2E 测试

---

## 11. License V2 API 验证

### 11.1 新增文件列表

```shell
# 基础设施层
ls -la internal/infrastructure/couchdb/
ls -la internal/infrastructure/repository/

# Handler 层
ls -la internal/interfaces/api/handler/handlelicense*.go
```

### 11.2 API 端点测试

```bash
# 1. 启动服务器
go run main.go serve

# 2. 激活 License
curl -X POST http://localhost:1314/api/license/v2/activate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "MDF-STARTER-TEST-1234",
    "device_id": "device-uuid-1234",
    "device_name": "MacBook Pro",
    "device_type": "desktop"
  }'

# 3. 查询 License 信息
curl "http://localhost:1314/api/license/v2/info?key=MDF-STARTER-TEST-1234"

# 4. 查询设备列表
curl "http://localhost:1314/api/license/v2/devices?key=MDF-STARTER-TEST-1234"

# 5. 查询 IP 列表
curl "http://localhost:1314/api/license/v2/ips?key=MDF-STARTER-TEST-1234"

# 6. 查询 Sync 信息
curl "http://localhost:1314/api/license/v2/sync?key=MDF-STARTER-TEST-1234"

# 7. 查询 Publish 信息
curl "http://localhost:1314/api/license/v2/publish?key=MDF-STARTER-TEST-1234"
```

### 11.3 运行新增测试

```bash
# CouchDB Client 测试
go test -v ./internal/infrastructure/couchdb/...

# License Handler 测试
go test -v ./internal/interfaces/api/handler/handlelicense_test.go \
  ./internal/interfaces/api/handler/handlelicense.go
```

