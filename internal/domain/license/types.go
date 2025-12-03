package license

import (
	"time"
)

// LicensePlan represents the type of license
type LicensePlan string

const (
	PlanFree     LicensePlan = "free"
	PlanYearly   LicensePlan = "yearly"
	PlanLifetime LicensePlan = "lifetime"
)

// LicensePayload represents the license payload structure
type LicensePayload struct {
	LicenseKey   string                `json:"licenseKey"`   // License Key (MDF-XXXX-XXXX-XXXX)
	DeviceID     string                `json:"deviceId"`     // 设备唯一标识
	Plan         LicensePlan           `json:"plan"`         // 授权类型
	Exp          *time.Time            `json:"exp"`          // 过期时间（永久版为null）
	ResourceKeys map[string]string     `json:"resourceKeys"` // 分级加密的CEK: {"basic": "encrypted_cek", "premium": "encrypted_cek"}
	IssueAt      time.Time             `json:"issueAt"`      // 签发时间
	Version      int                   `json:"version"`      // license schema版本
}

// License represents a complete license with payload and signature
type License struct {
	Payload   string `json:"payload"`   // Base64 encoded JSON payload
	Signature string `json:"signature"` // ECDSA signature
}

// KeyPair represents a cryptographic key pair
type KeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// PublicKeys represents the public keys needed by frontend
type PublicKeys struct {
	ECDSAPublicKey string `json:"ecdsaPublicKey"` // 用于验证license签名
	RSAPublicKey   string `json:"rsaPublicKey"`   // 用于解密resourceKey
}

// LicenseRequest represents a request to generate licenses
type LicenseRequest struct {
	Plan  LicensePlan `json:"plan"`
	Count int         `json:"count"` // 生成数量
}

// LicenseDetail represents detailed information about a license key
type LicenseDetail struct {
	LicenseKey       string      `json:"licenseKey"`       // License Key (MDF-XXXX-XXXX-XXXX)
	Plan             LicensePlan `json:"plan"`             // 授权类型
	IssueDate        string      `json:"issueDate"`        // 签发日期 (YYYY-MM-DD)
	ExpiryDate       string      `json:"expiryDate"`       // 过期日期 (YYYY-MM-DD, lifetime为9999-01-01)
	MaxActivations   int         `json:"maxActivations"`   // 最大激活数量
	CurrentActivations int       `json:"currentActivations"` // 当前激活数量
	DeviceIDs        []string          `json:"deviceIds"`        // 已激活的设备ID列表
	ResourceKeys     map[string]string `json:"resourceKeys"`     // 分级加密的CEK
	Version          int               `json:"version"`          // license schema版本
}

// LicenseRegistry represents the main license registry file
type LicenseRegistry struct {
	GeneratedAt  string   `json:"generatedAt"`  // 生成时间 (YYYY-MM-DD)
	Plan         string   `json:"plan"`         // 计划类型
	Count        int      `json:"count"`        // 生成数量
	LicenseKeys  []string `json:"licenseKeys"`  // license key 列表
}

// ActivationRequest represents a request to activate a license
type ActivationRequest struct {
	LicenseKey string `json:"licenseKey"` // License Key to activate
	DeviceID   string `json:"deviceId"`   // Device unique identifier
}

// ActivationResponse represents the response after license activation
type ActivationResponse struct {
	Success     bool           `json:"success"`
	License     *License       `json:"license,omitempty"`     // Activated license
	PublicKeys  *PublicKeys    `json:"publicKeys,omitempty"`  // Public keys for frontend
	ErrorMsg    string         `json:"errorMsg,omitempty"`    // Error message if activation failed
	Detail      *LicenseDetail `json:"detail,omitempty"`      // License detail info
}

// ContentEncryptionKey represents the CEK used to encrypt theme content
type ContentEncryptionKey []byte

// ContentLevel represents the access level of content
type ContentLevel string

const (
	ContentLevelBasic   ContentLevel = "basic"   // 基础内容，lifetime + yearly 都能访问
	ContentLevelPremium ContentLevel = "premium" // 高级内容，只有 yearly 能访问
)
