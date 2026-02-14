# 试用 License API - 快速参考

## 🎯 功能概述

开放 API，允许用户通过邮箱免费申请试用 License。

## 📡 API 端点

```
POST /api/license/trial
```

## 📝 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | ✅ | 申请人邮箱地址 |

## ✅ 成功响应

```json
{
  "data": [{
    "success": true,
    "license_key": "MDF-ABCD-EFGH-IJKL",
    "email": "abcd-efgh-ijkl@mdfriday.com",
    "password": "YWJjZC1lZmdoLWlqa2w=",
    "message": "Trial license created successfully. Please check your email for details.",
    "validity_days": 3
  }]
}
```

## ❌ 错误响应

### 邮箱已使用
```json
{
  "data": [{
    "success": false,
    "error": "This email has already been used to request a trial license"
  }]
}
```

### 邮箱格式错误
```json
{
  "data": [{
    "success": false,
    "error": "Invalid email format"
  }]
}
```

## 🔧 SMTP 配置（腾讯企业邮箱）

在 Admin CMS 系统配置中设置：

```
Host:     smtp.exmail.qq.com
Port:     465
Username: noreply@mdfriday.com
Password: ****************
From:     noreply@mdfriday.com
UseTLS:   ✓
```

## 📧 邮件模板

用户将收到包含以下信息的邮件：
- License Key
- Login Email
- Login Password  
- 有效期天数

## 🧪 测试命令

```bash
# 基本测试
curl -X POST http://localhost:1314/api/license/trial \
  -F "email=test@example.com"

# 带详细输出
curl -X POST http://localhost:1314/api/license/trial \
  -F "email=test@example.com" \
  -v
```

## 🛡️ 安全特性

- ✅ 邮箱格式验证（RFC 5322）
- ✅ 防止重复申请（邮箱唯一）
- ✅ TLS 加密传输（隐式 TLS）
- ✅ 随机 Key 生成

## 📊 数据流

```
用户提交邮箱
    ↓
验证邮箱格式
    ↓
检查是否重复
    ↓
生成 License
    ↓
创建用户账户
    ↓
发送邮件
    ↓
返回结果
```

## 💡 注意事项

1. **一个邮箱只能申请一次**试用
2. **SMTP 未配置时**仍会创建 License，但不发送邮件
3. **邮件发送失败**不影响 License 创建
4. **默认有效期**: 3 天（free plan）
5. **端口 465**: 隐式 TLS（腾讯企业邮箱推荐）

## 📚 相关文档

- [完整实现文档](../docs/trial-license-implementation.md)
- [SMTP 使用指南](../internal/infrastructure/smtp/README.md)
- [License Kit 文档](../internal/infrastructure/licensekit/)

