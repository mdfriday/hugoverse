# SMTP 邮件发送服务

本包提供了通用的 SMTP 邮件发送功能，支持两种 TLS 连接方式。

## 功能特性

- ✅ 支持隐式 TLS（465 端口）- 适用于腾讯企业邮箱
- ✅ 支持 STARTTLS（587 端口）- 适用于 Gmail, Outlook 等
- ✅ 支持纯文本和 HTML 邮件
- ✅ 自动根据端口选择合适的 TLS 连接方式
- ✅ 完整的错误处理和日志

## TLS 连接方式说明

### 隐式 TLS（465 端口）

**特点**：从连接开始就使用 TLS 加密

**适用于**：
- 腾讯企业邮箱（smtp.exmail.qq.com）
- QQ 邮箱（smtp.qq.com）
- 部分企业邮箱服务

**配置示例**：
```go
config := &smtp.Config{
    Host:     "smtp.exmail.qq.com",
    Port:     465,           // 隐式 TLS 端口
    Username: "your-email@your-domain.com",
    Password: "your-password",
    From:     "your-email@your-domain.com",
    UseTLS:   true,          // 启用 TLS，自动识别为隐式 TLS
}
```

### STARTTLS（587 端口）

**特点**：先建立明文连接，然后升级到 TLS

**适用于**：
- Gmail（smtp.gmail.com）
- Outlook（smtp-mail.outlook.com）
- 大多数国际邮件服务

**配置示例**：
```go
config := &smtp.Config{
    Host:     "smtp.gmail.com",
    Port:     587,           // STARTTLS 端口
    Username: "your-email@gmail.com",
    Password: "your-app-password",
    From:     "your-email@gmail.com",
    UseTLS:   true,          // 启用 TLS，自动识别为 STARTTLS
}
```

## 使用示例

### 发送纯文本邮件

```go
package main

import (
    "github.com/mdfriday/hugoverse/internal/infrastructure/smtp"
)

func main() {
    // 创建 SMTP 配置（腾讯企业邮箱示例）
    config := &smtp.Config{
        Host:     "smtp.exmail.qq.com",
        Port:     465,
        Username: "noreply@mdfriday.com",
        Password: "your-smtp-password",
        From:     "noreply@mdfriday.com",
        UseTLS:   true,
    }

    // 创建 SMTP 客户端
    client := smtp.NewClient(config)

    // 构建邮件
    msg := &smtp.EmailMessage{
        To:      []string{"user@example.com"},
        Subject: "Welcome to MDFriday",
        Body:    "Thank you for signing up!",
        IsHTML:  false,
    }

    // 发送邮件
    err := client.SendEmail(msg)
    if err != nil {
        log.Fatalf("Failed to send email: %v", err)
    }

    log.Println("Email sent successfully!")
}
```

### 发送 HTML 邮件

```go
htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Welcome to MDFriday!</h1>
    <p>Your trial license: <strong>MDF-XXXX-XXXX-XXXX</strong></p>
</body>
</html>
`

msg := &smtp.EmailMessage{
    To:      []string{"user@example.com"},
    Subject: "Your MDFriday Trial License",
    Body:    htmlBody,
    IsHTML:  true, // 设置为 HTML 格式
}

err := client.SendEmail(msg)
```

### 发送给多个收件人

```go
msg := &smtp.EmailMessage{
    To:      []string{"user1@example.com", "user2@example.com", "user3@example.com"},
    Subject: "Announcement",
    Body:    "This is an important announcement.",
    IsHTML:  false,
}

err := client.SendEmail(msg)
```

## 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| Host | string | ✅ | SMTP 服务器地址（如 smtp.exmail.qq.com） |
| Port | int | ✅ | SMTP 端口（465 或 587） |
| Username | string | ✅ | SMTP 用户名（通常是完整邮箱地址） |
| Password | string | ✅ | SMTP 密码或应用专用密码 |
| From | string | ✅ | 发件人邮箱地址 |
| UseTLS | bool | ✅ | 是否使用 TLS（true = 启用 TLS） |

## 端口选择指南

| 端口 | 连接方式 | 适用场景 | 推荐度 |
|------|----------|----------|--------|
| 465 | 隐式 TLS | 腾讯企业邮箱、QQ 邮箱 | ⭐⭐⭐⭐⭐ |
| 587 | STARTTLS | Gmail、Outlook、大多数邮件服务 | ⭐⭐⭐⭐⭐ |
| 25 | 明文/可选 TLS | 不推荐（不安全） | ❌ |

## 常见邮件服务配置

### 腾讯企业邮箱（推荐）

```go
Host:     "smtp.exmail.qq.com"
Port:     465
UseTLS:   true
```

### QQ 邮箱

```go
Host:     "smtp.qq.com"
Port:     465  或  587
UseTLS:   true
```

### Gmail

```go
Host:     "smtp.gmail.com"
Port:     587
UseTLS:   true
// 注意：需要使用应用专用密码，不是 Google 账户密码
```

### Outlook/Hotmail

```go
Host:     "smtp-mail.outlook.com"
Port:     587
UseTLS:   true
```

### 163 邮箱

```go
Host:     "smtp.163.com"
Port:     465  或  994
UseTLS:   true
```

## 错误处理

```go
err := client.SendEmail(msg)
if err != nil {
    // 常见错误类型：
    // - "failed to establish TLS connection" - TLS 连接失败
    // - "failed to authenticate" - 认证失败（检查用户名密码）
    // - "failed to set recipient" - 收件人地址无效
    // - "failed to write email body" - 邮件内容写入失败
    
    log.Printf("Failed to send email: %v", err)
    return
}
```

## 配置验证

在发送邮件前，可以先验证配置是否正确：

```go
config := &smtp.Config{
    Host:     "smtp.exmail.qq.com",
    Port:     465,
    Username: "noreply@mdfriday.com",
    Password: "your-password",
    From:     "noreply@mdfriday.com",
    UseTLS:   true,
}

// 验证配置
err := smtp.ValidateConfig(config)
if err != nil {
    log.Fatalf("Invalid SMTP config: %v", err)
}
```

## 注意事项

1. **应用专用密码**：Gmail 等服务需要使用应用专用密码，而不是账户密码
2. **防火墙设置**：确保服务器可以访问 SMTP 端口（465 或 587）
3. **发件人地址**：From 字段必须与 Username 匹配（大多数 SMTP 服务器要求）
4. **腾讯企业邮箱**：建议使用 465 端口的隐式 TLS
5. **邮件内容**：使用 UTF-8 编码，支持中文

## 测试

运行测试（需要配置真实的 SMTP 凭据）：

```bash
go test ./internal/infrastructure/smtp/... -v
```

## 在 Trial License API 中的使用

在 `handletrial.go` 中的使用示例：

```go
if s.adminApp.SMTP.IsConfigured() {
    smtpClient := smtp.NewClient(&smtp.Config{
        Host:     s.adminApp.SMTP.Host(),
        Port:     s.adminApp.SMTP.Port(),     // 465 for 腾讯企业邮箱
        Username: s.adminApp.SMTP.Username(),
        Password: s.adminApp.SMTP.Password(),
        From:     s.adminApp.SMTP.From(),
        UseTLS:   s.adminApp.SMTP.UseTLS(),   // true
    })

    emailMsg := &smtp.EmailMessage{
        To:      []string{email},
        Subject: "Your MDFriday Trial License",
        Body:    emailBody,
        IsHTML:  false,
    }

    err = smtpClient.SendEmail(emailMsg)
    if err != nil {
        log.Errorf("Failed to send email: %v", err)
    }
}
```

## 实现原理

### 隐式 TLS（465 端口）

1. 使用 `tls.Dial()` 直接建立 TLS 连接
2. 在已加密的连接上创建 SMTP 客户端
3. 执行 SMTP 协议（EHLO、AUTH、MAIL、RCPT、DATA）

### STARTTLS（587 端口）

1. 使用 `smtp.Dial()` 建立明文连接
2. 发送 STARTTLS 命令升级到 TLS
3. 在加密连接上执行 SMTP 认证和发送

## 自动选择逻辑

代码会根据端口号自动选择合适的连接方式：

- `Port == 465` → 使用隐式 TLS（`sendWithImplicitTLS`）
- `Port == 587` → 使用 STARTTLS（`sendWithSTARTTLS`）
- `UseTLS == false` → 使用明文连接（不推荐）

## License

MIT License

