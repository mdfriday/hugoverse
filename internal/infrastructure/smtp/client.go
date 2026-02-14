package smtp

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Config SMTP 配置
type Config struct {
	Host     string // SMTP 服务器地址，例如：smtp.qq.com
	Port     int    // SMTP 端口，例如：587 (STARTTLS) 或 465 (隐式 TLS)
	Username string // SMTP 用户名（通常是邮箱地址）
	Password string // SMTP 密码或应用专用密码
	From     string // 发件人邮箱地址
	UseTLS   bool   // 是否使用 TLS（true = 使用 TLS，根据端口自动判断 STARTTLS 或隐式 TLS）
}

// Client SMTP 客户端
type Client struct {
	config *Config
}

// NewClient 创建 SMTP 客户端
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
	}
}

// EmailMessage 邮件消息
type EmailMessage struct {
	To      []string // 收件人列表
	Subject string   // 邮件主题
	Body    string   // 邮件正文（纯文本）
	IsHTML  bool     // 是否为 HTML 格式
}

// SendEmail 发送邮件
func (c *Client) SendEmail(msg *EmailMessage) error {
	if c.config == nil {
		return fmt.Errorf("SMTP config is not set")
	}

	if len(msg.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// 构建邮件内容
	body := c.buildEmailBody(msg)

	// SMTP 服务器地址
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// 认证信息
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	// 发送邮件
	if c.config.UseTLS {
		// 根据端口判断使用隐式 TLS 还是 STARTTLS
		if c.config.Port == 465 {
			// 465 端口使用隐式 TLS（从一开始就是 TLS 连接）
			return c.sendWithImplicitTLS(addr, auth, msg.To, body)
		} else {
			// 587 端口使用 STARTTLS（先明文连接，再升级到 TLS）
			return c.sendWithSTARTTLS(addr, auth, msg.To, body)
		}
	} else {
		// 使用标准 SMTP（不推荐，不安全）
		return smtp.SendMail(addr, auth, c.config.From, msg.To, []byte(body))
	}
}

// sendWithImplicitTLS 使用隐式 TLS 发送邮件（465 端口）
// 从一开始就建立 TLS 连接，适用于腾讯企业邮箱等
func (c *Client) sendWithImplicitTLS(addr string, auth smtp.Auth, to []string, body string) error {
	// 直接建立 TLS 连接
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to establish TLS connection: %w", err)
	}
	defer conn.Close()

	// 创建 SMTP 客户端
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// 发送 EHLO/HELO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("failed to send HELLO: %w", err)
	}

	// 认证
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// 设置发件人
	if err := client.Mail(c.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// 发送邮件正文
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// sendWithSTARTTLS 使用 STARTTLS 发送邮件（587 端口）
// 先明文连接，再升级到 TLS
func (c *Client) sendWithSTARTTLS(addr string, auth smtp.Auth, to []string, body string) error {
	// 连接到 SMTP 服务器
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// 发送 EHLO/HELO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("failed to send HELLO: %w", err)
	}

	// 启动 TLS
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// 认证
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// 设置发件人
	if err := client.Mail(c.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// 发送邮件正文
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// buildEmailBody 构建邮件正文
func (c *Client) buildEmailBody(msg *EmailMessage) string {
	var builder strings.Builder

	// 邮件头部
	builder.WriteString(fmt.Sprintf("From: %s\r\n", c.config.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// Content-Type
	if msg.IsHTML {
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	builder.WriteString("\r\n")

	// 邮件正文
	builder.WriteString(msg.Body)

	return builder.String()
}

// ValidateConfig 验证 SMTP 配置
func ValidateConfig(config *Config) error {
	if config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if config.Port == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if config.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if config.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}
	if config.From == "" {
		return fmt.Errorf("SMTP from address is required")
	}

	return nil
}

