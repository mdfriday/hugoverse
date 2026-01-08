package caddy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DomainCheckResult 域名检查结果
type DomainCheckResult struct {
	Domain        string   `json:"domain"`
	DNSValid      bool     `json:"dns_valid"`
	ResolvedIPs   []string `json:"resolved_ips"`
	HTTPReachable bool     `json:"http_reachable"`
	Error         string   `json:"error,omitempty"`
	Ready         bool     `json:"ready"`
}

// DomainChecker 域名检查器
type DomainChecker struct {
	ServerIP    string
	DNSTimeout  time.Duration
	HTTPTimeout time.Duration
}

// NewDomainChecker 创建域名检查器
func NewDomainChecker(serverIP string) *DomainChecker {
	return &DomainChecker{
		ServerIP:    serverIP,
		DNSTimeout:  10 * time.Second,
		HTTPTimeout: 15 * time.Second,
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

// CheckHTTP 检查 HTTP 可达性
// 验证域名的 80 端口是否可以访问（用于 HTTP-01 challenge）
func (c *DomainChecker) CheckHTTP(domain string) *DomainCheckResult {
	result := &DomainCheckResult{
		Domain: domain,
	}

	// 创建带超时的 HTTP 客户端
	client := &http.Client{
		Timeout: c.HTTPTimeout,
		// 不跟随重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 尝试访问 ACME challenge 路径
	// 即使返回 404 也说明 HTTP 可达
	url := fmt.Sprintf("http://%s/.well-known/acme-challenge/test", domain)
	
	resp, err := client.Get(url)
	if err != nil {
		result.Error = fmt.Sprintf("HTTP check failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// 任何 HTTP 响应都说明可达（包括 404）
	// Let's Encrypt 只需要能访问到服务器即可
	result.HTTPReachable = true
	return result
}

// CheckAll 执行完整的域名检查
// 依次检查 DNS 和 HTTP，全部通过则 Ready=true
func (c *DomainChecker) CheckAll(domain string) *DomainCheckResult {
	// 先检查 DNS
	dnsResult := c.CheckDNS(domain)
	if !dnsResult.DNSValid {
		return dnsResult
	}

	// 再检查 HTTP
	httpResult := c.CheckHTTP(domain)
	
	// 合并结果
	result := &DomainCheckResult{
		Domain:        domain,
		DNSValid:      dnsResult.DNSValid,
		ResolvedIPs:   dnsResult.ResolvedIPs,
		HTTPReachable: httpResult.HTTPReachable,
	}

	if !httpResult.HTTPReachable {
		result.Error = httpResult.Error
		return result
	}

	// 全部通过
	result.Ready = true
	return result
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

