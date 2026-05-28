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
// 同时支持腾讯云 DNS (tencentcloud) 与阿里云 DNS (alidns)
// 凭据直接在配置中传递（环境变量方式在某些 Caddy 版本中不工作）
// 字段命名规则因插件而异：
//   - tencentcloud: PascalCase（SecretId / SecretKey）
//   - alidns:       snake_case（access_key_id / access_key_secret）
//
// 通过 BuildDNSProvider 工厂按 provider 名称填充对应字段，其它字段保持 omitempty。
type DNSProvider struct {
	Name string `json:"name"`

	// tencentcloud (DNSPod)
	SecretId     string `json:"SecretId,omitempty"`
	SecretKey    string `json:"SecretKey,omitempty"`
	Region       string `json:"Region,omitempty"`
	SessionToken string `json:"SessionToken,omitempty"`

	// alidns (Alibaba Cloud DNS)
	AccessKeyID     string `json:"access_key_id,omitempty"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	SecurityToken   string `json:"security_token,omitempty"`
}

// DNS Provider 名称常量
const (
	DNSProviderTencentCloud = "tencentcloud"
	DNSProviderAliDNS       = "alidns"
)

// BuildDNSProvider 根据 provider 名称把通用 (id, secret) 翻译成对应插件需要的字段
// 未识别的 provider 名称会回退到 tencentcloud，保持向后兼容。
func BuildDNSProvider(providerName, id, secret string) *DNSProvider {
	switch providerName {
	case DNSProviderAliDNS:
		return &DNSProvider{
			Name:            DNSProviderAliDNS,
			AccessKeyID:     id,
			AccessKeySecret: secret,
		}
	case DNSProviderTencentCloud, "":
		fallthrough
	default:
		return &DNSProvider{
			Name:      DNSProviderTencentCloud,
			SecretId:  id,
			SecretKey: secret,
		}
	}
}

// HTTPChallenge HTTP-01 Challenge 配置
type HTTPChallenge struct {
	// 空结构表示使用默认 HTTP-01
}

// NewDNS01Policy 创建 DNS-01 策略（用于平台域名的 Wildcard 证书）
//
// 前提条件：
// 使用 DNS-01 challenge 需要 Caddy 在构建时包含对应的 DNS provider 插件。
// 当前 docker/caddy/Dockerfile 同时编入了：
//
//	xcaddy build --with github.com/caddy-dns/tencentcloud \
//	             --with github.com/caddy-dns/alidns
//
// 因此 provider name 可以是 "tencentcloud" 或 "alidns"。
//
// 注意：
//   - 凭据语义随 provider 变化：tencentcloud 是 (SecretId, SecretKey)，alidns 是 (AccessKeyID, AccessKeySecret)。
//   - 字段如何序列化到 JSON 由 BuildDNSProvider 决定，调用方只需提供通用 (id, secret) 即可。
//   - DNSPod 已被腾讯云收购，原 github.com/caddy-dns/dnspod 已过时；统一使用 caddy-dns/tencentcloud。
//   - 如果使用官方预编译的 Caddy 二进制文件，DNS-01 challenge 将无法工作。
func NewDNS01Policy(id string, subjects []string, providerName, credID, credSecret string) AutomationPolicy {
	return AutomationPolicy{
		ID:       id,
		Subjects: subjects,
		Issuers: []Issuer{
			{
				Module: "acme",
				Challenges: &ChallengeConfig{
					DNS: &DNSChallenge{
						Provider: BuildDNSProvider(providerName, credID, credSecret),
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
// 凭据直接在配置中传递，credID/credSecret 的语义随 providerName 而定。
func NewWildcardDNS01Policy(coreDomain, providerName, credID, credSecret string) AutomationPolicy {
	subjects := []string{coreDomain, "*." + coreDomain}
	return NewDNS01Policy("platform-wildcard", subjects, providerName, credID, credSecret)
}

// ==================== TLS 配置生成函数 ====================

// GeneratePlatformTLSConfig 生成平台域名的 TLS 配置
// 用于启动时配置 Wildcard 证书（mdfriday.com 和 *.mdfriday.com）
// 所有 subdomain 将自动使用此 Wildcard 证书，无需单独申请
//
// 前提条件：
//  1. 需要构建包含对应 DNS 插件（tencentcloud 或 alidns）的 Caddy 实例（参见 NewDNS01Policy 注释）
//  2. 凭据直接在配置中传递；providerName 为空时默认走 tencentcloud（向后兼容）
func GeneratePlatformTLSConfig(coreDomain, providerName, credID, credSecret string) *TLSConfig {
	if coreDomain == "" || credID == "" || credSecret == "" {
		return nil
	}
	if providerName == "" {
		providerName = DNSProviderTencentCloud
	}

	wildcardPolicy := NewWildcardDNS01Policy(coreDomain, providerName, credID, credSecret)

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

