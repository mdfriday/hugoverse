# Auto-Init 改进说明（License 检查版本）

## 改进概述

在之前的改进基础上，进一步增强了初始化检查逻辑：通过检查 **License 数量**来精确判断企业功能是否已配置，而不仅仅依赖用户存在性。

## 问题分析

### 原始问题

1. ❌ 每次重启容器都会执行企业功能配置
2. ❌ 每次重启都会尝试生成新的企业 License
3. ❌ 数据库中有重复的 License 记录

### 改进前的检查方式

```go
if db.SystemInitComplete() {  // 仅检查是否有用户
    // 跳过或执行企业功能配置
}
```

**局限性：**
- 只能判断系统是否初始化（是否有用户）
- 无法判断企业功能是否已配置
- 如果企业功能配置失败，重启后无法自动修复

### 改进后的检查方式

```go
if db.SystemInitComplete() {
    licenseCount := contentApp.GetLicenseCount()
    hasLicense := licenseCount > 0
    
    if hasLicense {
        // 已配置，跳过
    } else {
        // 未配置，尝试配置
    }
}
```

**优势：**
- ✅ 更精确地判断企业功能状态
- ✅ 能检测到配置失败的情况
- ✅ 支持自动修复（重启后自动重试）

## 核心修改

### 1. License 实体新增方法（`internal/domain/content/entity/license.go`）

```go
// GetLicenseCount 获取 License 总数
// 用于判断是否已经生成过企业 License
func (c *Content) GetLicenseCount() int {
    allLicenses := c.Repo.AllContent("License")
    return len(allLicenses)
}

// HasAnyLicense 检查是否存在任何 License
// 用于判断企业功能是否已配置
func (c *Content) HasAnyLicense() bool {
    return c.GetLicenseCount() > 0
}
```

### 2. AutoInitialize 函数签名更新（`internal/application/auto_init.go`）

```go
// LicenseChecker 接口定义
type LicenseChecker interface {
    GetLicenseCount() int
    HasAnyLicense() bool
}

// AutoInitialize 从环境变量自动初始化系统
func AutoInitialize(adminApp *entity.Admin, db SystemDatabase, contentApp LicenseChecker, log loggers.Logger) error {
    // ...
}
```

### 3. 增强的初始化检查逻辑

```go
if db.SystemInitComplete() {
    log.Println("✅ System already initialized")
    
    // 检查企业功能是否已配置（通过 License 数量判断）
    licenseCount := contentApp.GetLicenseCount()
    hasLicense := licenseCount > 0
    
    if hasLicense {
        log.Printf("ℹ️  Enterprise features already configured (found %d license(s))", licenseCount)
    }
    
    // 检查是否需要强制重新配置企业功能
    forceReconfigure := os.Getenv("FORCE_RECONFIGURE_ENTERPRISE") == "true"
    
    if forceReconfigure {
        log.Println("⚠️  FORCE_RECONFIGURE_ENTERPRISE=true detected")
        log.Println("   Re-configuring enterprise features...")
        go delayedEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log)
    } else if !hasLicense {
        // 系统已初始化，但没有 License，可能是企业功能配置失败
        // 尝试重新配置企业功能
        log.Println("⚠️  No licenses found, enterprise features may not be configured")
        log.Println("   Attempting to configure enterprise features...")
        go delayedEnterpriseFeaturesWithLicenseCheck(adminApp, contentApp, log)
    } else {
        // 系统已初始化且有 License，跳过企业功能配置
        log.Println("   Skipping enterprise features configuration on restart")
        log.Println("   💡 To force reconfigure: set FORCE_RECONFIGURE_ENTERPRISE=true")
    }

    return nil
}
```

### 4. 企业功能配置添加 License 检查

```go
func configureEnterpriseFeaturesWithLicenseCheck(adminApp *entity.Admin, contentApp LicenseChecker, log loggers.Logger) error {
    // 1. 初始化 Caddy 路由
    // 2. 自动配置企业站点
    
    // 3. 检查 License 是否已存在
    licenseCount := contentApp.GetLicenseCount()
    if licenseCount > 0 {
        log.Printf("ℹ️  Found %d existing license(s), skipping license generation", licenseCount)
        log.Println("   💡 To force regenerate: set FORCE_RECONFIGURE_ENTERPRISE=true")
    } else {
        // 4. 自动生成企业 License（如果启用且不存在）
        if os.Getenv("AUTO_GENERATE_ENTERPRISE_LICENSE") == "true" {
            if err := generateEnterpriseLicense(log); err != nil {
                // ...
            }
        }
    }
    
    return nil
}
```

### 5. 更新调用方（`internal/interfaces/api/server.go`）

```go
// 尝试自动初始化（在启动后台任务之前）
// 传入 contentApp 以便检查 License 数量
if err := application.AutoInitialize(s.adminApp, s.db, contentApp, s.Log); err != nil {
    s.Log.Warnf("Auto-initialization failed: %v", err)
    s.Log.Println("Please visit /admin/init to configure manually")
}
```

## 使用场景对比

### 场景 1：首次启动

```bash
docker-compose --env-file .env.local up -d
```

**行为：**
1. ✅ 系统未初始化 → 创建管理员
2. ✅ License 数量 = 0 → 配置企业功能
3. ✅ 生成企业 License

**日志：**
```
🔧 AUTO_INIT=true, initializing...
👤 Creating admin user: admin@localhost
💾 Saving system configuration
✅ System initialization completed!
⏳ Waiting for server to start...
🏢 Configuring Enterprise Features
📋 Enterprise License Generation
   [1/1] Creating: HGVS-XXXX-XXXX-XXXX
      ✅ User created
      ✅ License created
✅ Enterprise Features Configuration Complete
```

### 场景 2：正常重启（有 License）

```bash
docker-compose --env-file .env.local restart
```

**行为：**
1. ✅ 系统已初始化
2. ✅ License 数量 = 1 → **跳过**企业功能配置
3. ✅ 直接启动服务

**日志：**
```
✅ System already initialized
ℹ️  Enterprise features already configured (found 1 license(s))
   Skipping enterprise features configuration on restart
   💡 To force reconfigure: set FORCE_RECONFIGURE_ENTERPRISE=true
```

### 场景 3：异常重启（无 License）

假设首次启动时 License 生成失败，重启后：

**行为：**
1. ✅ 系统已初始化
2. ⚠️ License 数量 = 0 → **自动修复**
3. ✅ 重新配置企业功能

**日志：**
```
✅ System already initialized
⚠️  No licenses found, enterprise features may not be configured
   Attempting to configure enterprise features...
⏳ Waiting for server to start...
🏢 Configuring Enterprise Features
📋 Enterprise License Generation
   [1/1] Creating: HGVS-XXXX-XXXX-XXXX
      ✅ User created
      ✅ License created
✅ Enterprise Features Configuration Complete
```

### 场景 4：强制重新配置

```bash
# .env.local
FORCE_RECONFIGURE_ENTERPRISE=true
```

**行为：**
1. ✅ 系统已初始化
2. ✅ License 数量 = 1
3. ⚠️ `FORCE_RECONFIGURE_ENTERPRISE=true` → 强制重新配置
4. ⏭️ 已存在的 License 会被跳过（幂等性）

**日志：**
```
✅ System already initialized
ℹ️  Enterprise features already configured (found 1 license(s))
⚠️  FORCE_RECONFIGURE_ENTERPRISE=true detected
   Re-configuring enterprise features...
⏳ Waiting for server to start...
🏢 Configuring Enterprise Features
ℹ️  Found 1 existing license(s), skipping license generation
   💡 To force regenerate: set FORCE_RECONFIGURE_ENTERPRISE=true
✅ Enterprise Features Configuration Complete
```

## 决策流程图

```
启动
 │
 ├─► 系统未初始化？
 │    ├─ 是 → 执行初始化 → 配置企业功能 → 生成 License
 │    └─ 否 ↓
 │
 ├─► 检查 License 数量
 │    │
 │    ├─► License > 0？
 │    │    ├─ 是 → 检查 FORCE_RECONFIGURE_ENTERPRISE
 │    │    │        ├─ true → 重新配置（跳过已有 License）
 │    │    │        └─ false → 跳过配置
 │    │    │
 │    │    └─ 否 → 自动修复：重新配置企业功能
 │
 └─► 启动服务
```

## 核心优势

### 1. 精确检测

- ✅ 不仅检查系统是否初始化
- ✅ 还检查企业功能是否已配置
- ✅ 基于实际的 License 数据

### 2. 自动修复

- ✅ 检测到配置不完整时自动修复
- ✅ 首次启动失败后重启能自动补全
- ✅ 不需要手动干预

### 3. 防止重复

- ✅ 有 License 时跳过配置
- ✅ 避免重复生成
- ✅ 保持数据库清洁

### 4. 灵活控制

- ✅ 支持强制重新配置
- ✅ 支持跳过已有 License
- ✅ 清晰的日志输出

## 接口设计

### LicenseChecker 接口

```go
type LicenseChecker interface {
    GetLicenseCount() int    // 获取 License 总数
    HasAnyLicense() bool     // 是否存在任何 License
}
```

**实现者：**
- `*Content` 实体（`internal/domain/content/entity/content.go`）

**使用者：**
- `AutoInitialize` 函数
- `configureEnterpriseFeaturesWithLicenseCheck` 函数

## 向后兼容性

### 保留的旧函数

```go
// delayedEnterpriseFeatures 已废弃，保留以便向后兼容
func delayedEnterpriseFeatures(adminApp *entity.Admin, log loggers.Logger) {
    // ...
}

// configureEnterpriseFeatures 已废弃，保留以便向后兼容
func configureEnterpriseFeatures(adminApp *entity.Admin, log loggers.Logger) error {
    // ...
}
```

### 新函数

```go
// 推荐使用
func delayedEnterpriseFeaturesWithLicenseCheck(adminApp *entity.Admin, contentApp LicenseChecker, log loggers.Logger) {
    // ...
}

func configureEnterpriseFeaturesWithLicenseCheck(adminApp *entity.Admin, contentApp LicenseChecker, log loggers.Logger) error {
    // ...
}
```

## 测试验证

### 1. 验证首次启动

```bash
docker-compose --env-file .env.local down -v
rm -rf ./data/hugoverse/.system*
docker-compose --env-file .env.local up -d
docker-compose --env-file .env.local logs -f hugoverse | grep -A5 "License"
```

**预期：**
- 看到 License 生成流程
- 成功创建 1 个 License

### 2. 验证正常重启

```bash
docker-compose --env-file .env.local restart hugoverse
docker-compose --env-file .env.local logs -f hugoverse | grep -A2 "already configured"
```

**预期：**
- 看到 "Enterprise features already configured (found 1 license(s))"
- 看到 "Skipping enterprise features configuration"

### 3. 验证自动修复

```bash
# 模拟：手动删除 License 数据
docker exec -it hugoverse-app sh
# 在容器内删除 License 相关数据（具体命令取决于存储方式）

# 重启
docker-compose --env-file .env.local restart hugoverse
docker-compose --env-file .env.local logs -f hugoverse
```

**预期：**
- 看到 "No licenses found, enterprise features may not be configured"
- 看到 "Attempting to configure enterprise features"
- 重新生成 License

### 4. 验证 License 计数

```bash
# 在容器内
docker exec -it hugoverse-app sh
# 使用数据库工具查询 License 数量
```

**预期：**
- 首次启动后：1 个 License
- 多次重启后：仍然是 1 个 License（不增加）

## 相关文件

### 修改的文件

1. `internal/domain/content/entity/license.go`
   - 新增 `GetLicenseCount()` 方法
   - 新增 `HasAnyLicense()` 方法

2. `internal/application/auto_init.go`
   - 新增 `LicenseChecker` 接口
   - 修改 `AutoInitialize()` 函数签名
   - 新增 `delayedEnterpriseFeaturesWithLicenseCheck()` 函数
   - 新增 `configureEnterpriseFeaturesWithLicenseCheck()` 函数
   - 增强初始化检查逻辑

3. `internal/interfaces/api/server.go`
   - 更新 `AutoInitialize()` 调用，传入 `contentApp`

### 文档

- `.mdfriday/auto-init-improvements.md` - 基础改进说明
- `.mdfriday/auto-init-license-check.md` - 本文档（License 检查版本）

## 环境变量

| 变量名 | 说明 | 默认值 | 备注 |
|--------|------|--------|------|
| `AUTO_INIT` | 是否启用自动初始化 | `false` | 首次启动必需 |
| `AUTO_GENERATE_ENTERPRISE_LICENSE` | 是否自动生成 License | `true` | - |
| `AUTO_CONFIGURE_ENTERPRISE_SITE` | 是否自动配置企业站点 | `true` | - |
| `FORCE_RECONFIGURE_ENTERPRISE` | 强制重新配置 | `false` | 调试/修复用 |
| `ENTERPRISE_LICENSE_KEY` | 指定 License Key | - | 可选 |
| `ENTERPRISE_LICENSE_COUNT` | 生成 License 数量 | `1` | - |

## 总结

### 改进前

- ❌ 只检查用户存在性
- ❌ 无法判断企业功能状态
- ❌ 配置失败后需手动修复
- ❌ 可能产生重复 License

### 改进后

- ✅ 检查 License 数量（精确判断）
- ✅ 能判断企业功能是否已配置
- ✅ 自动检测并修复配置问题
- ✅ 防止重复生成 License
- ✅ 支持强制重新配置
- ✅ 清晰的日志输出
- ✅ 向后兼容

### 用户体验

- 🚀 更智能的启动检查
- 🔧 自动修复配置问题
- 🔒 更干净的数据（无重复）
- 💡 灵活的配置选项
- 📝 更详细的状态反馈
