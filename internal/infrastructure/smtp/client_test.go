package smtp_test

import (
	"testing"

	"github.com/mdfriday/hugoverse/internal/infrastructure/smtp"
)

// TestImplicitTLS 测试隐式 TLS 连接（465 端口，适用于腾讯企业邮箱）
func TestImplicitTLS(t *testing.T) {
	t.Skip("Skip integration test - requires real SMTP credentials")

	// 腾讯企业邮箱配置示例
	config := &smtp.Config{
		Host:     "smtp.exmail.qq.com", // 腾讯企业邮箱 SMTP 服务器
		Port:     465,                   // 隐式 TLS 端口
		Username: "your-email@your-domain.com",
		Password: "your-password-or-app-specific-password",
		From:     "your-email@your-domain.com",
		UseTLS:   true, // 启用 TLS，会自动根据端口选择隐式 TLS
	}

	client := smtp.NewClient(config)

	msg := &smtp.EmailMessage{
		To:      []string{"recipient@example.com"},
		Subject: "Test Email via Implicit TLS (465)",
		Body:    "This is a test email sent via implicit TLS connection on port 465.",
		IsHTML:  false,
	}

	err := client.SendEmail(msg)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	t.Log("Email sent successfully via implicit TLS (465)")
}

// TestSTARTTLS 测试 STARTTLS 连接（587 端口）
func TestSTARTTLS(t *testing.T) {
	t.Skip("Skip integration test - requires real SMTP credentials")

	// STARTTLS 配置示例（Gmail, Outlook 等）
	config := &smtp.Config{
		Host:     "smtp.gmail.com",
		Port:     587, // STARTTLS 端口
		Username: "your-email@gmail.com",
		Password: "your-app-specific-password",
		From:     "your-email@gmail.com",
		UseTLS:   true, // 启用 TLS，会自动根据端口选择 STARTTLS
	}

	client := smtp.NewClient(config)

	msg := &smtp.EmailMessage{
		To:      []string{"recipient@example.com"},
		Subject: "Test Email via STARTTLS (587)",
		Body:    "This is a test email sent via STARTTLS connection on port 587.",
		IsHTML:  false,
	}

	err := client.SendEmail(msg)
	if err != nil {
		t.Fatalf("Failed to send email: %v", err)
	}

	t.Log("Email sent successfully via STARTTLS (587)")
}

// TestHTMLEmail 测试 HTML 邮件
func TestHTMLEmail(t *testing.T) {
	t.Skip("Skip integration test - requires real SMTP credentials")

	config := &smtp.Config{
		Host:     "smtp.exmail.qq.com",
		Port:     465,
		Username: "your-email@your-domain.com",
		Password: "your-password",
		From:     "your-email@your-domain.com",
		UseTLS:   true,
	}

	client := smtp.NewClient(config)

	htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Welcome to MDFriday!</h1>
    <p>Your trial license has been created successfully.</p>
    <div style="background-color: #f0f0f0; padding: 15px; margin: 20px 0;">
        <p><strong>License Key:</strong> MDF-XXXX-XXXX-XXXX</p>
        <p><strong>Email:</strong> xxxx-xxxx-xxxx@mdfriday.com</p>
        <p><strong>Password:</strong> xxxxxxxx</p>
    </div>
    <p>Thank you for choosing MDFriday!</p>
</body>
</html>
`

	msg := &smtp.EmailMessage{
		To:      []string{"recipient@example.com"},
		Subject: "Your MDFriday Trial License",
		Body:    htmlBody,
		IsHTML:  true,
	}

	err := client.SendEmail(msg)
	if err != nil {
		t.Fatalf("Failed to send HTML email: %v", err)
	}

	t.Log("HTML email sent successfully")
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *smtp.Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &smtp.Config{
				Host:     "smtp.example.com",
				Port:     465,
				Username: "user@example.com",
				Password: "password",
				From:     "user@example.com",
				UseTLS:   true,
			},
			wantErr: false,
		},
		{
			name: "Missing host",
			config: &smtp.Config{
				Port:     465,
				Username: "user@example.com",
				Password: "password",
				From:     "user@example.com",
				UseTLS:   true,
			},
			wantErr: true,
		},
		{
			name: "Missing port",
			config: &smtp.Config{
				Host:     "smtp.example.com",
				Username: "user@example.com",
				Password: "password",
				From:     "user@example.com",
				UseTLS:   true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smtp.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

