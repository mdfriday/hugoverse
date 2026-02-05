package caddy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// DomainCheckResult 域名检查结果
type DomainCheckResult struct {
	Domain      string            `json:"domain"`
	DNSValid    bool              `json:"dns_valid"`
	ResolvedIPs []string          `json:"resolved_ips"`
	TLSReady    bool              `json:"tls_ready"`
	TLSStatus   string            `json:"tls_status"`       // dns_pending, cert_pending, cert_error, active
	CertInfo    *CertificateInfo  `json:"certificate,omitempty"`
	Error       string            `json:"error,omitempty"`
	Ready       bool              `json:"ready"`
}

// DomainChecker 域名检查器
type DomainChecker struct {
	ServerIP   string
	DNSTimeout time.Duration
	TLSTimeout time.Duration
}

// NewDomainChecker 创建域名检查器
func NewDomainChecker(serverIP string) *DomainChecker {
	return &DomainChecker{
		ServerIP:   serverIP,
		DNSTimeout: 10 * time.Second,
		TLSTimeout: 15 * time.Second,
	}
}

// CheckDNS 检查域名 DNS 解析
// 验证域名是否解析到指定的服务器 IP
func (c *DomainChecker) CheckDNS(domain string) *DomainCheckResult {
	result := &DomainCheckResult{
		Domain:      domain,
		ResolvedIPs: []string{},
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), c.DNSTimeout)
	defer cancel()

	// 创建自定义 resolver
	resolver := &net.Resolver{
		PreferGo: true,
	}

	// 解析域名
	ips, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		result.Error = fmt.Sprintf("DNS lookup failed: %v", err)
		return result
	}

	result.ResolvedIPs = ips

	// 检查是否包含服务器 IP
	if c.ServerIP == "" {
		// 如果没有配置 ServerIP，只要能解析就算通过
		result.DNSValid = len(ips) > 0
		if !result.DNSValid {
			result.Error = "Domain does not resolve to any IP address"
		}
		return result
	}

	// 检查解析的 IP 是否包含服务器 IP
	for _, ip := range ips {
		if ip == c.ServerIP {
			result.DNSValid = true
			return result
		}
	}

	result.Error = fmt.Sprintf("DNS does not point to server IP %s, found: %v", c.ServerIP, ips)
	return result
}

// CheckTLS 检查 HTTPS TLS 握手和证书有效性
// 这是判断 HTTPS 是否就绪的【主判据】
func (c *DomainChecker) CheckTLS(domain string) *DomainCheckResult {
	result := &DomainCheckResult{
		Domain:    domain,
		TLSStatus: "cert_pending",
	}

	// 创建带超时的 dialer
	dialer := &net.Dialer{
		Timeout: c.TLSTimeout,
	}

	// 尝试 TLS 连接
	conn, err := tls.DialWithDialer(dialer, "tcp", domain+":443", &tls.Config{
		ServerName: domain,           // SNI
		MinVersion: tls.VersionTLS12,
	})

	if err != nil {
		// TLS 握手失败（证书未签发或网络问题）
		result.Error = fmt.Sprintf("TLS handshake failed: %v", err)
		return result
	}
	defer conn.Close()

	// 获取证书
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		result.TLSStatus = "cert_error"
		result.Error = "No certificate presented by server"
		return result
	}

	cert := state.PeerCertificates[0]

	// 验证证书有效期
	now := time.Now()
	if now.Before(cert.NotBefore) {
		result.TLSStatus = "cert_error"
		result.Error = fmt.Sprintf("Certificate not yet valid (valid from %s)", cert.NotBefore)
		return result
	}
	if now.After(cert.NotAfter) {
		result.TLSStatus = "cert_error"
		result.Error = fmt.Sprintf("Certificate expired (expired on %s)", cert.NotAfter)
		return result
	}

	// 验证主机名匹配
	if err := cert.VerifyHostname(domain); err != nil {
		result.TLSStatus = "cert_error"
		result.Error = fmt.Sprintf("Certificate hostname mismatch: %v", err)
		return result
	}

	// ✅ TLS 握手成功，证书有效
	result.TLSReady = true
	result.TLSStatus = "active"
	result.Ready = true
	result.CertInfo = &CertificateInfo{
		Status:     "issued",
		Issuer:     cert.Issuer.CommonName,
		Subject:    cert.Subject.CommonName,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		DNSNames:   cert.DNSNames,
		IsWildcard: strings.HasPrefix(cert.Subject.CommonName, "*."),
	}

	return result
}

// CheckAll 执行域名检查（仅 DNS）
// 注意：不包含 TLS 检测（TLS 检测在用户查询状态时进行）
func (c *DomainChecker) CheckAll(domain string) *DomainCheckResult {
	// 检查 DNS
	dnsResult := c.CheckDNS(domain)
	if !dnsResult.DNSValid {
		dnsResult.TLSStatus = "dns_pending"
		return dnsResult
	}

	// DNS 就绪
	dnsResult.Ready = true
	dnsResult.TLSStatus = "dns_ready"
	return dnsResult
}

// CheckDNSOnly 仅检查 DNS（不检查 HTTP）
// 用于快速验证 DNS 配置
func (c *DomainChecker) CheckDNSOnly(domain string) *DomainCheckResult {
	result := c.CheckDNS(domain)
	if result.DNSValid {
		result.Ready = true
	}
	return result
}

