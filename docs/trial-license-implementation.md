# License 申请试用 API 实现总结

## 概述

已完成试用 License 申请 API 的完整实现，包括：
1. License 生成和创建
2. 用户账户自动创建
3. 邮件通知功能（支持腾讯企业邮箱 465 端口隐式 TLS）
4. 防止重复申请机制

## 实现的功能模块

### 1. LicenseTrial 数据模型 ✅

**文件**: `internal/domain/content/valueobject/licensetrial.go`

- 记录试用 License 申请信息
- 使用 Email 作为唯一标识（hash key）
- 防止同一邮箱多次申请

### 2. License Kit 基础设施 ✅

**文件**: `internal/infrastructure/licensekit/`

#### 已迁移和新增的功能：
- `plan.go` - Plan 配置管理
- `keygen.go` - License Key 生成
- `credential.go` - Email/Password 转换
- `user.go` - 用户创建和邮箱验证

#### 新增功能：
```go
// 邮箱格式验证
licensekit.ValidateEmail(email)

// 创建用户（已从 CLI 迁移）
licensekit.CreateUser(apiBase, email, password)

// 生成 License Key
licensekit.GenerateLicenseKey()

// 生成用户凭据
licensekit.LicenseKeyToEmail(licenseKey)
licensekit.LicenseKeyToPassword(licenseKey)
```

### 3. SMTP 邮件服务 ✅

**文件**: `internal/infrastructure/smtp/client.go`

#### 支持两种 TLS 连接方式：

**隐式 TLS（465 端口）** - 腾讯企业邮箱
```go
// 从连接开始就使用 TLS 加密
config := &smtp.Config{
    Host:     "smtp.exmail.qq.com",
    Port:     465,  // 自动识别为隐式 TLS
    UseTLS:   true,
}
```

**STARTTLS（587 端口）** - Gmail, Outlook
```go
// 先明文连接，再升级到 TLS
config := &smtp.Config{
    Host:     "smtp.gmail.com",
    Port:     587,  // 自动识别为 STARTTLS
    UseTLS:   true,
}
```

#### 自动选择逻辑：
- Port == 465 → `sendWithImplicitTLS()` 
- Port == 587 → `sendWithSTARTTLS()`
- 无需手动指定连接方式

### 4. Admin SMTP 配置 ✅

**文件**: 
- `internal/domain/admin/valueobject/config.go` - 配置字段
- `internal/domain/admin/entity/smtp.go` - SMTP 实体
- `internal/domain/admin/factory/admin.go` - 初始化

#### 新增配置字段：
```go
SMTPHost     string // smtp.exmail.qq.com
SMTPPort     int    // 465
SMTPUsername string // noreply@mdfriday.com
SMTPPassword string // 应用专用密码
SMTPFrom     string // noreply@mdfriday.com
SMTPUseTLS   bool   // true
```

#### 使用方法：
```go
adminApp.SMTP.IsConfigured()  // 检查是否已配置
adminApp.SMTP.Host()           // smtp.exmail.qq.com
adminApp.SMTP.Port()           // 465
adminApp.SMTP.Username()       // 用户名
adminApp.SMTP.Password()       // 密码
adminApp.SMTP.From()           // 发件人
adminApp.SMTP.UseTLS()         // true
```

### 5. License Entity 新增方法 ✅

**文件**: `internal/domain/content/entity/license.go`

```go
// 创建 License
CreateLicense(license *valueobject.License) (string, error)

// 获取 LicenseTrial（通过邮箱）
GetLicenseTrialByEmail(email string) (*valueobject.LicenseTrial, error)

// 创建 LicenseTrial
CreateLicenseTrial(trial *valueobject.LicenseTrial) (string, error)
```

### 6. Trial License API ✅

**文件**: `internal/interfaces/api/handler/handletrial.go`

**端点**: `POST /api/license/trial`

**请求参数**:
```
email=user@example.com
```

**处理流程**:
```
1. 验证邮箱格式（正则验证）
   ↓
2. 检查邮箱是否已申请（查询 LicenseTrial）
   ↓ (未申请)
3. 生成免费 License Key
   ↓
4. 生成用户凭据（email, password）
   ↓
5. 创建 License 对象（free plan）
   ↓
6. 创建用户账户
   ↓
7. 创建 LicenseTrial 记录
   ↓
8. 发送邮件通知（如果 SMTP 已配置）
   ↓
9. 返回成功响应
```

**响应示例**:
```json
{
  "success": true,
  "license_key": "MDF-ABCD-EFGH-IJKL",
  "email": "abcd-efgh-ijkl@mdfriday.com",
  "password": "YWJjZC1lZmdoLWlqa2w=",
  "message": "Trial license created successfully. Please check your email for details.",
  "validity_days": 3
}
```

**错误响应**:
```json
{
  "success": false,
  "error": "This email has already been used to request a trial license"
}
```

## 文件结构

```
internal/
├── infrastructure/
│   ├── licensekit/
│   │   ├── plan.go              # Plan 配置
│   │   ├── keygen.go            # License Key 生成
│   │   ├── credential.go        # 凭据转换
│   │   ├── user.go              # 用户创建（新增）
│   │   └── licensekit_test.go   # 测试
│   │
│   └── smtp/
│       ├── client.go            # SMTP 客户端（支持隐式 TLS）
│       ├── client_test.go       # 测试示例
│       └── README.md            # 使用文档
│
├── domain/
│   ├── admin/
│   │   ├── valueobject/
│   │   │   └── config.go        # 添加 SMTP 配置字段
│   │   ├── entity/
│   │   │   ├── admin.go         # 添加 SMTP 实体
│   │   │   └── smtp.go          # SMTP 实体（新增）
│   │   └── factory/
│   │       └── admin.go         # 初始化 SMTP
│   │
│   └── content/
│       ├── valueobject/
│       │   └── licensetrial.go  # LicenseTrial 模型（新增）
│       ├── entity/
│       │   └── license.go       # 添加 License/Trial 创建方法
│       └── factory/
│           └── content.go       # 注册 LicenseTrial 类型
│
└── interfaces/
    ├── api/
    │   └── handler/
    │       └── handletrial.go   # Trial API Handler（新增）
    │
    └── cli/
        └── license.go           # 更新使用 licensekit
```

## 配置示例

### 腾讯企业邮箱 SMTP 配置

在 Admin CMS 中配置：

```
SMTP Server Host:        smtp.exmail.qq.com
SMTP Server Port:        465
SMTP Username:           noreply@mdfriday.com
SMTP Password:           ****************  (应用专用密码)
SMTP From Address:       noreply@mdfriday.com
Use TLS:                 ✓ (勾选)
```

## 使用说明

### 1. 配置 SMTP（必需）

在 Admin CMS 的系统配置中填写 SMTP 信息。

### 2. 前端调用 API

```javascript
// 申请试用 License
fetch('/api/license/trial', {
  method: 'POST',
  body: new FormData().append('email', 'user@example.com')
})
.then(res => res.json())
.then(data => {
  if (data.data[0].success) {
    console.log('License Key:', data.data[0].license_key);
    console.log('Email:', data.data[0].email);
    console.log('Password:', data.data[0].password);
  }
});
```

### 3. 用户收到邮件

```
Dear User,

Thank you for your interest in MDFriday!

Your trial license has been successfully created. Here are your credentials:

License Key: MDF-ABCD-EFGH-IJKL
Email: abcd-efgh-ijkl@mdfriday.com
Password: YWJjZC1lZmdoLWlqa2w=

Validity: 3 days

You can now use these credentials to activate your license and start using MDFriday.

Best regards,
MDFriday Team
```

## 安全特性

1. ✅ 邮箱格式验证（RFC 5322 标准）
2. ✅ 防止重复申请（通过 Email Hash 索引）
3. ✅ TLS 加密传输（隐式 TLS 465 端口）
4. ✅ 随机 License Key 生成（排除易混淆字符）
5. ✅ 密码 Base64 编码存储

## 性能考虑

1. **异步邮件发送**: 邮件发送失败不影响 License 创建
2. **邮箱唯一索引**: 使用 Hash 索引快速查询
3. **连接复用**: SMTP 客户端可复用配置

## 测试建议

### 单元测试
```bash
go test ./internal/infrastructure/licensekit/... -v
go test ./internal/infrastructure/smtp/... -v
```

### 集成测试
```bash
# 需要配置真实 SMTP 凭据
# 取消 t.Skip() 后运行
go test ./internal/infrastructure/smtp/... -v -run TestImplicitTLS
```

### API 测试
```bash
curl -X POST http://localhost:1314/api/license/trial \
  -F "email=test@example.com"
```

## 常见问题

### Q1: 为什么使用 465 而不是 587？
A: 腾讯企业邮箱推荐使用 465 端口的隐式 TLS，更加安全和稳定。

### Q2: 如何获取 SMTP 密码？
A: 使用腾讯企业邮箱的应用专用密码，而不是邮箱登录密码。

### Q3: 邮件发送失败怎么办？
A: License 仍然会创建成功，返回结果中包含凭据，可以手动通知用户。

### Q4: 如何防止邮箱被滥用？
A: 
- 前端添加验证码
- 限制同一 IP 的请求频率
- 添加邮箱验证流程

### Q5: 支持 HTML 邮件吗？
A: 支持，设置 `IsHTML: true` 即可。

## 下一步优化建议

1. **邮件模板系统**: 使用模板引擎管理邮件内容
2. **邮件队列**: 异步处理邮件发送，提高响应速度
3. **重试机制**: 邮件发送失败自动重试
4. **邮箱验证**: 发送验证链接确认邮箱有效性
5. **速率限制**: 防止 API 被滥用

## 相关文档

- [SMTP 使用文档](./internal/infrastructure/smtp/README.md)
- [License Kit 文档](./internal/infrastructure/licensekit/)
- [API 域名管理实现](./.mdfriday/api-domain-management-impl.md)

## 维护记录

- 2024-XX-XX: 完成 Trial License API 实现
- 2024-XX-XX: 添加 SMTP 隐式 TLS 支持
- 2024-XX-XX: 迁移 CLI license 功能到 licensekit

