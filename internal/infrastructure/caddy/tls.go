package caddy

// TLSConfig TLS App 配置
type TLSConfig struct {
	Automation *AutomationConfig `json:"automation,omitempty"`
}

// AutomationConfig 自动证书管理配置
type AutomationConfig struct {
	Policies []AutomationPolicy `json:"policies,omitempty"`
}

// AutomationPolicy 证书策略
type AutomationPolicy struct {
	ID       string   `json:"@id,omitempty"`
	Subjects []string `json:"subjects"`
	Issuers  []Issuer `json:"issuers,omitempty"`
}

// Issuer 证书签发器配置
type Issuer struct {
	Module     string           `json:"module"`
	Challenges *ChallengeConfig `json:"challenges,omitempty"`
}

// ChallengeConfig ACME Challenge 配置
type ChallengeConfig struct {
	DNS  *DNSChallenge  `json:"dns,omitempty"`
	HTTP *HTTPChallenge `json:"http,omitempty"`
}

// DNSChallenge DNS-01 Challenge 配置
type DNSChallenge struct {
	Provider *DNSProvider `json:"provider"`
}

// DNSProvider DNS 提供商配置
// 腾讯云 DNS (tencentcloud) 凭据通过环境变量传递：
// - TENCENTCLOUD_SECRET_ID
// - TENCENTCLOUD_SECRET_KEY
type DNSProvider struct {
	Name string `json:"name"`
}

// HTTPChallenge HTTP-01 Challenge 配置
type HTTPChallenge struct {
	// 空结构表示使用默认 HTTP-01
}

// NewDNS01Policy 创建 DNS-01 策略（用于平台域名的 Wildcard 证书）
//
// 前提条件：
// 使用 DNS-01 challenge 需要 Caddy 在构建时包含对应的 DNS provider 插件。
// 对于腾讯云 DNS（DNSPod），需要使用 xcaddy 构建包含 caddy-dns/tencentcloud 插件的 Caddy 实例：
//
//	# 在 Ubuntu 上安装 xcaddy
//	go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
//
//	# 构建包含腾讯云 DNS 插件的 Caddy
//	xcaddy build --with github.com/caddy-dns/tencentcloud
//
//	# 构建后的 caddy 二进制文件将支持 tencentcloud DNS-01 challenge
//
// 注意：DNSPod 已被腾讯云收购，原 github.com/caddy-dns/dnspod 已过时且不兼容新版 libdns。
// 请使用 github.com/caddy-dns/tencentcloud，provider name 为 "tencentcloud"。
// 凭据通过环境变量传递：TENCENTCLOUD_SECRET_ID 和 TENCENTCLOUD_SECRET_KEY
//
// 如果使用官方预编译的 Caddy 二进制文件，DNS-01 challenge 将无法工作。
func NewDNS01Policy(id string, subjects []string, providerName string) AutomationPolicy {
	return AutomationPolicy{
		ID:       id,
		Subjects: subjects,
		Issuers: []Issuer{
			{
				Module: "acme",
				Challenges: &ChallengeConfig{
					DNS: &DNSChallenge{
						Provider: &DNSProvider{
							Name: providerName,
						},
					},
				},
			},
		},
	}
}

// NewHTTP01Policy 创建 HTTP-01 策略（用于用户自定义域名）
func NewHTTP01Policy(id string, subjects []string) AutomationPolicy {
	return AutomationPolicy{
		ID:       id,
		Subjects: subjects,
		Issuers: []Issuer{
			{
				Module: "acme",
				Challenges: &ChallengeConfig{
					HTTP: &HTTPChallenge{},
				},
			},
		},
	}
}

// NewSingleDomainHTTP01Policy 创建单域名 HTTP-01 策略
// 用于用户自定义域名（每个 License 最多一个自定义域名）
func NewSingleDomainHTTP01Policy(domain string) AutomationPolicy {
	policyID := "custom-" + sanitizeDomainForID(domain)
	return NewHTTP01Policy(policyID, []string{domain})
}

// NewWildcardDNS01Policy 创建 Wildcard DNS-01 策略
// 用于平台域名（如 mdfriday.com 和 *.mdfriday.com）
//
// 前提条件：需要构建包含 DNS provider 插件的 Caddy 实例（参见 NewDNS01Policy 注释）
// 凭据通过环境变量传递：TENCENTCLOUD_SECRET_ID 和 TENCENTCLOUD_SECRET_KEY
func NewWildcardDNS01Policy(coreDomain, providerName string) AutomationPolicy {
	subjects := []string{coreDomain, "*." + coreDomain}
	return NewDNS01Policy("platform-wildcard", subjects, providerName)
}

// ==================== TLS 配置生成函数 ====================

// GeneratePlatformTLSConfig 生成平台域名的 TLS 配置
// 用于启动时配置 Wildcard 证书（mdfriday.com 和 *.mdfriday.com）
// 所有 subdomain 将自动使用此 Wildcard 证书，无需单独申请
//
// 前提条件：
// 1. 需要构建包含腾讯云 DNS 插件的 Caddy 实例（参见 NewDNS01Policy 注释）
// 2. 凭据通过环境变量传递：TENCENTCLOUD_SECRET_ID 和 TENCENTCLOUD_SECRET_KEY
func GeneratePlatformTLSConfig(coreDomain string) *TLSConfig {
	if coreDomain == "" {
		return nil
	}

	// 创建平台域名的 Wildcard 策略（使用 tencentcloud provider）
	// 凭据通过环境变量传递给 Caddy 进程
	wildcardPolicy := NewWildcardDNS01Policy(coreDomain, "tencentcloud")

	return &TLSConfig{
		Automation: &AutomationConfig{
			Policies: []AutomationPolicy{wildcardPolicy},
		},
	}
}

// IsPlatformDomain 判断域名是否为平台域名（使用 Wildcard 证书）
// 平台域名包括：coreDomain 本身，以及 *.coreDomain 的所有子域名
// 例如：mdfriday.com, user123.mdfriday.com, cdb.mdfriday.com 都是平台域名
func IsPlatformDomain(domain, coreDomain string) bool {
	if domain == "" || coreDomain == "" {
		return false
	}
	
	// 完全匹配
	if domain == coreDomain {
		return true
	}
	
	// 子域名匹配（以 .coreDomain 结尾）
	suffix := "." + coreDomain
	if len(domain) > len(suffix) && domain[len(domain)-len(suffix):] == suffix {
		return true
	}
	
	return false
}

// NeedsSeparateTLSPolicy 判断域名是否需要单独的 TLS 策略
// 只有非平台域名（用户自定义域名）才需要单独的 HTTP-01 策略
func NeedsSeparateTLSPolicy(domain, coreDomain string) bool {
	return !IsPlatformDomain(domain, coreDomain)
}

