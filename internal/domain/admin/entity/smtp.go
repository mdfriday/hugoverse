package entity

import "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"

// SMTP SMTP 配置实体
type SMTP struct {
	Conf *valueobject.Config
}

// Host 返回 SMTP 服务器地址
func (s *SMTP) Host() string {
	if s.Conf == nil {
		return ""
	}
	return s.Conf.SMTPHost
}

// Port 返回 SMTP 端口
func (s *SMTP) Port() int {
	if s.Conf == nil {
		return 587 // 默认 TLS 端口
	}
	if s.Conf.SMTPPort == 0 {
		return 587
	}
	return s.Conf.SMTPPort
}

// Username 返回 SMTP 用户名
func (s *SMTP) Username() string {
	if s.Conf == nil {
		return ""
	}
	return s.Conf.SMTPUsername
}

// Password 返回 SMTP 密码
func (s *SMTP) Password() string {
	if s.Conf == nil {
		return ""
	}
	return s.Conf.SMTPPassword
}

// From 返回发件人邮箱地址
func (s *SMTP) From() string {
	if s.Conf == nil {
		return ""
	}
	return s.Conf.SMTPFrom
}

// UseTLS 返回是否使用 TLS
func (s *SMTP) UseTLS() bool {
	if s.Conf == nil {
		return true // 默认启用 TLS
	}
	return s.Conf.SMTPUseTLS
}

// IsConfigured 检查 SMTP 是否已配置
func (s *SMTP) IsConfigured() bool {
	return s.Host() != "" && s.Username() != "" && s.Password() != "" && s.From() != ""
}

