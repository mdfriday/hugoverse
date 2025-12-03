package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// CryptoService handles all cryptographic operations for the license system
type CryptoService struct {
	ecdsaPrivateKey    *ecdsa.PrivateKey
	ecdsaPublicKey     *ecdsa.PublicKey
	rsaPrivateKey      *rsa.PrivateKey  // 用于包装 license 中的 CEK
	rsaPublicKey       *rsa.PublicKey
	basicKEKPrivateKey *rsa.PrivateKey  // KEK_BASIC: 用于加密基础内容
	basicKEKPublicKey  *rsa.PublicKey
	premiumKEKPrivateKey *rsa.PrivateKey // KEK_PREMIUM: 用于加密高级内容
	premiumKEKPublicKey  *rsa.PublicKey
}

// NewCryptoService creates a new crypto service with generated keys
func NewCryptoService() (*CryptoService, error) {
	// Generate ECDSA key pair for signing
	ecdsaPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	// Generate RSA key pair for encryption
	rsaPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Generate KEK_BASIC key pair
	basicKEKPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate basic KEK: %w", err)
	}

	// Generate KEK_PREMIUM key pair
	premiumKEKPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate premium KEK: %w", err)
	}

	return &CryptoService{
		ecdsaPrivateKey:      ecdsaPrivKey,
		ecdsaPublicKey:       &ecdsaPrivKey.PublicKey,
		rsaPrivateKey:        rsaPrivKey,
		rsaPublicKey:         &rsaPrivKey.PublicKey,
		basicKEKPrivateKey:   basicKEKPrivKey,
		basicKEKPublicKey:    &basicKEKPrivKey.PublicKey,
		premiumKEKPrivateKey: premiumKEKPrivKey,
		premiumKEKPublicKey:  &premiumKEKPrivKey.PublicKey,
	}, nil
}

// LoadCryptoService loads crypto service from existing keys
func LoadCryptoService(ecdsaPrivKeyPEM, rsaPrivKeyPEM, basicKEKPrivKeyPEM, premiumKEKPrivKeyPEM string) (*CryptoService, error) {
	// Parse ECDSA private key
	ecdsaBlock, _ := pem.Decode([]byte(ecdsaPrivKeyPEM))
	if ecdsaBlock == nil {
		return nil, errors.New("failed to decode ECDSA private key PEM")
	}

	ecdsaPrivKey, err := x509.ParseECPrivateKey(ecdsaBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}

	// Parse RSA private key
	rsaBlock, _ := pem.Decode([]byte(rsaPrivKeyPEM))
	if rsaBlock == nil {
		return nil, errors.New("failed to decode RSA private key PEM")
	}

	rsaPrivKey, err := x509.ParsePKCS1PrivateKey(rsaBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	// Parse Basic KEK private key
	basicKEKBlock, _ := pem.Decode([]byte(basicKEKPrivKeyPEM))
	if basicKEKBlock == nil {
		return nil, errors.New("failed to decode basic KEK private key PEM")
	}

	basicKEKPrivKey, err := x509.ParsePKCS1PrivateKey(basicKEKBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse basic KEK private key: %w", err)
	}

	// Parse Premium KEK private key
	premiumKEKBlock, _ := pem.Decode([]byte(premiumKEKPrivKeyPEM))
	if premiumKEKBlock == nil {
		return nil, errors.New("failed to decode premium KEK private key PEM")
	}

	premiumKEKPrivKey, err := x509.ParsePKCS1PrivateKey(premiumKEKBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse premium KEK private key: %w", err)
	}

	return &CryptoService{
		ecdsaPrivateKey:      ecdsaPrivKey,
		ecdsaPublicKey:       &ecdsaPrivKey.PublicKey,
		rsaPrivateKey:        rsaPrivKey,
		rsaPublicKey:         &rsaPrivKey.PublicKey,
		basicKEKPrivateKey:   basicKEKPrivKey,
		basicKEKPublicKey:    &basicKEKPrivKey.PublicKey,
		premiumKEKPrivateKey: premiumKEKPrivKey,
		premiumKEKPublicKey:  &premiumKEKPrivKey.PublicKey,
	}, nil
}

// GetECDSAKeyPair returns the ECDSA key pair in PEM format
func (cs *CryptoService) GetECDSAKeyPair() (*KeyPair, error) {
	// Private key
	privKeyBytes, err := x509.MarshalECPrivateKey(cs.ecdsaPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ECDSA private key: %w", err)
	}

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cs.ecdsaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ECDSA public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return &KeyPair{
		PrivateKey: string(privKeyPEM),
		PublicKey:  string(pubKeyPEM),
	}, nil
}

// GetRSAKeyPair returns the RSA key pair in PEM format
func (cs *CryptoService) GetRSAKeyPair() (*KeyPair, error) {
	// Private key
	privKeyBytes := x509.MarshalPKCS1PrivateKey(cs.rsaPrivateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cs.rsaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RSA public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return &KeyPair{
		PrivateKey: string(privKeyPEM),
		PublicKey:  string(pubKeyPEM),
	}, nil
}

// GetBasicKEKKeyPair returns the basic KEK key pair in PEM format
func (cs *CryptoService) GetBasicKEKKeyPair() (*KeyPair, error) {
	// Private key
	privKeyBytes := x509.MarshalPKCS1PrivateKey(cs.basicKEKPrivateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cs.basicKEKPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal basic KEK public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return &KeyPair{
		PrivateKey: string(privKeyPEM),
		PublicKey:  string(pubKeyPEM),
	}, nil
}

// GetPremiumKEKKeyPair returns the premium KEK key pair in PEM format
func (cs *CryptoService) GetPremiumKEKKeyPair() (*KeyPair, error) {
	// Private key
	privKeyBytes := x509.MarshalPKCS1PrivateKey(cs.premiumKEKPrivateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cs.premiumKEKPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal premium KEK public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return &KeyPair{
		PrivateKey: string(privKeyPEM),
		PublicKey:  string(pubKeyPEM),
	}, nil
}

// GetPublicKeys returns the public keys needed by frontend
func (cs *CryptoService) GetPublicKeys() (*PublicKeys, error) {
	ecdsaKeyPair, err := cs.GetECDSAKeyPair()
	if err != nil {
		return nil, err
	}

	rsaKeyPair, err := cs.GetRSAKeyPair()
	if err != nil {
		return nil, err
	}

	return &PublicKeys{
		ECDSAPublicKey: ecdsaKeyPair.PublicKey,
		RSAPublicKey:   rsaKeyPair.PublicKey,
	}, nil
}

// GenerateCEK generates a new Content Encryption Key
func (cs *CryptoService) GenerateCEK() (ContentEncryptionKey, error) {
	cek := make([]byte, 32) // AES-256
	_, err := rand.Read(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CEK: %w", err)
	}
	return ContentEncryptionKey(cek), nil
}

// WrapCEK encrypts the CEK using RSA-OAEP and returns base64 encoded resourceKey
func (cs *CryptoService) WrapCEK(cek ContentEncryptionKey) (string, error) {
	encryptedCEK, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, cs.rsaPublicKey, cek, nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt CEK: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encryptedCEK), nil
}

// UnwrapCEK decrypts the resourceKey to get the CEK
func (cs *CryptoService) UnwrapCEK(resourceKey string) (ContentEncryptionKey, error) {
	encryptedCEK, err := base64.StdEncoding.DecodeString(resourceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode resource key: %w", err)
	}

	cek, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, cs.rsaPrivateKey, encryptedCEK, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt CEK: %w", err)
	}

	return ContentEncryptionKey(cek), nil
}

// SignPayload signs the license payload using ECDSA
func (cs *CryptoService) SignPayload(payload *LicensePayload) (string, error) {
	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Hash the payload
	hash := sha256.Sum256(payloadBytes)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, cs.ecdsaPrivateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign payload: %w", err)
	}

	// Encode signature as base64 (pad to 32 bytes each)
	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	s.FillBytes(sBytes)
	signature := append(rBytes, sBytes...)
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifies the license signature using ECDSA public key
func (cs *CryptoService) VerifySignature(payload *LicensePayload, signature string) (bool, error) {
	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Hash the payload
	hash := sha256.Sum256(payloadBytes)

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Split signature into r and s
	if len(sigBytes) != 64 { // 32 bytes each for r and s
		return false, errors.New("invalid signature length")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Verify signature
	return ecdsa.Verify(cs.ecdsaPublicKey, hash[:], r, s), nil
}

// EncryptContent encrypts content using AES-GCM with the given CEK
func (cs *CryptoService) EncryptContent(cek ContentEncryptionKey, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// EncryptContentWithLevel encrypts content using the appropriate KEK based on content level
func (cs *CryptoService) EncryptContentWithLevel(content []byte, level ContentLevel) ([]byte, error) {
	// Generate CEK for this content
	cek, err := cs.GenerateCEK()
	if err != nil {
		return nil, fmt.Errorf("failed to generate CEK: %w", err)
	}

	// Encrypt content with CEK
	encryptedContent, err := cs.EncryptContent(cek, content)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt content: %w", err)
	}

	// Encrypt CEK with appropriate KEK
	var encryptedCEK []byte
	switch level {
	case ContentLevelBasic:
		encryptedCEK, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, cs.basicKEKPublicKey, cek, nil)
	case ContentLevelPremium:
		encryptedCEK, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, cs.premiumKEKPublicKey, cek, nil)
	default:
		return nil, fmt.Errorf("invalid content level: %s", level)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt CEK with KEK: %w", err)
	}

	// Combine level info + encrypted CEK + encrypted content
	levelBytes := []byte(level)
	result := make([]byte, 1+len(levelBytes)+4+len(encryptedCEK)+len(encryptedContent))
	
	// First byte: level length
	result[0] = byte(len(levelBytes))
	
	// Next: level string
	copy(result[1:], levelBytes)
	offset := 1 + len(levelBytes)
	
	// Next 4 bytes: encrypted CEK length
	result[offset] = byte(len(encryptedCEK) >> 24)
	result[offset+1] = byte(len(encryptedCEK) >> 16)
	result[offset+2] = byte(len(encryptedCEK) >> 8)
	result[offset+3] = byte(len(encryptedCEK))
	offset += 4
	
	// Next: encrypted CEK
	copy(result[offset:], encryptedCEK)
	offset += len(encryptedCEK)
	
	// Rest: encrypted content
	copy(result[offset:], encryptedContent)

	return result, nil
}

// DecryptContentWithLevel decrypts content that was encrypted with EncryptContentWithLevel
func (cs *CryptoService) DecryptContentWithLevel(encryptedData []byte, availableKEKs map[ContentLevel]*rsa.PrivateKey) ([]byte, error) {
	if len(encryptedData) < 1 {
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}
	
	// Extract level length
	levelLen := int(encryptedData[0])
	if len(encryptedData) < 1+levelLen+4 {
		return nil, fmt.Errorf("invalid encrypted data: insufficient length for level")
	}
	
	// Extract level
	levelBytes := encryptedData[1 : 1+levelLen]
	level := ContentLevel(levelBytes)
	offset := 1 + levelLen
	
	// Check if we have the required KEK
	kekPrivKey, exists := availableKEKs[level]
	if !exists {
		return nil, fmt.Errorf("access denied: license does not have permission to decrypt %s content", level)
	}
	
	// Extract encrypted CEK length
	cekLen := int(encryptedData[offset])<<24 | int(encryptedData[offset+1])<<16 | int(encryptedData[offset+2])<<8 | int(encryptedData[offset+3])
	offset += 4
	
	if len(encryptedData) < offset+cekLen {
		return nil, fmt.Errorf("invalid encrypted data: insufficient length for CEK")
	}
	
	// Extract encrypted CEK
	encryptedCEK := encryptedData[offset : offset+cekLen]
	offset += cekLen
	
	// Extract encrypted content
	encryptedContent := encryptedData[offset:]
	
	// Decrypt CEK using the appropriate KEK
	cek, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, kekPrivKey, encryptedCEK, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt CEK: %w", err)
	}
	
	// Decrypt content using CEK
	decryptedContent, err := cs.DecryptContent(ContentEncryptionKey(cek), encryptedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt content: %w", err)
	}
	
	return decryptedContent, nil
}

// DecryptContent decrypts content using AES-GCM with the given CEK
func (cs *CryptoService) DecryptContent(cek ContentEncryptionKey, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateLicenseKey creates a license key in MDF-XXXX-XXXX-XXXX format
func (cs *CryptoService) GenerateLicenseKey() (string, error) {
	// Generate 12 random characters (excluding confusing ones like 0, O, I, l)
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	key := make([]byte, 12)
	
	for i := range key {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		key[i] = chars[randomIndex.Int64()]
	}
	
	// Format as MDF-XXXX-XXXX-XXXX
	keyStr := string(key)
	return fmt.Sprintf("MDF-%s-%s-%s", keyStr[0:4], keyStr[4:8], keyStr[8:12]), nil
}

// GenerateLicense creates a complete license for the given request
func (cs *CryptoService) GenerateLicense(licenseKey string, plan LicensePlan) (*License, error) {
	return cs.GenerateLicenseWithDevice(licenseKey, plan, "")
}

// GenerateLicenseWithDevice creates a complete license with device binding
func (cs *CryptoService) GenerateLicenseWithDevice(licenseKey string, plan LicensePlan, deviceID string) (*License, error) {
	// Generate CEK
	cek, err := cs.GenerateCEK()
	if err != nil {
		return nil, err
	}

	// Wrap CEK with RSA
	resourceKey, err := cs.WrapCEK(cek)
	if err != nil {
		return nil, err
	}

	// Create payload with basic resource key (for backward compatibility)
	resourceKeys := map[string]string{
		"basic": resourceKey,
	}
	
	payload := &LicensePayload{
		LicenseKey:   licenseKey,
		DeviceID:     deviceID,
		Plan:         plan,
		ResourceKeys: resourceKeys,
		IssueAt:      time.Now(),
		Version:      1,
	}

	// Set expiration for yearly plans
	if plan == PlanYearly {
		exp := time.Now().AddDate(1, 0, 0) // 1 year from now
		payload.Exp = &exp
	}

	// Sign payload
	signature, err := cs.SignPayload(payload)
	if err != nil {
		return nil, err
	}

	// Encode payload as base64 JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &License{
		Payload:   base64.StdEncoding.EncodeToString(payloadBytes),
		Signature: signature,
	}, nil
}

// GenerateLicenseDetails creates multiple license details for the given request
func (cs *CryptoService) GenerateLicenseDetails(req *LicenseRequest) ([]*LicenseDetail, error) {
	var details []*LicenseDetail
	
	for i := 0; i < req.Count; i++ {
		// Generate license key
		licenseKey, err := cs.GenerateLicenseKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate license key %d: %w", i+1, err)
		}
		
		// Generate resource keys based on plan
		resourceKeys := make(map[string]string)
		
		// All plans get basic access
		basicCEK, err := cs.GenerateCEK()
		if err != nil {
			return nil, fmt.Errorf("failed to generate basic CEK for key %d: %w", i+1, err)
		}
		
		basicResourceKey, err := cs.WrapCEK(basicCEK)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap basic CEK for key %d: %w", i+1, err)
		}
		resourceKeys["basic"] = basicResourceKey
		
		// Yearly plans also get premium access
		if req.Plan == PlanYearly {
			premiumCEK, err := cs.GenerateCEK()
			if err != nil {
				return nil, fmt.Errorf("failed to generate premium CEK for key %d: %w", i+1, err)
			}
			
			premiumResourceKey, err := cs.WrapCEK(premiumCEK)
			if err != nil {
				return nil, fmt.Errorf("failed to wrap premium CEK for key %d: %w", i+1, err)
			}
			resourceKeys["premium"] = premiumResourceKey
		}
		
		// Create license detail
		issueDate := time.Now().Format("2006-01-02")
		var expiryDate string
		
		if req.Plan == PlanLifetime {
			expiryDate = "9999-01-01"
		} else if req.Plan == PlanYearly {
			expiryDate = time.Now().AddDate(1, 0, 0).Format("2006-01-02")
		} else {
			// Free plan - 30 days
			expiryDate = time.Now().AddDate(0, 0, 30).Format("2006-01-02")
		}
		
		detail := &LicenseDetail{
			LicenseKey:         licenseKey,
			Plan:               req.Plan,
			IssueDate:          issueDate,
			ExpiryDate:         expiryDate,
			MaxActivations:     3, // 默认最多3台设备
			CurrentActivations: 0,
			DeviceIDs:          []string{},
			ResourceKeys:       resourceKeys,
			Version:            1,
		}
		
		details = append(details, detail)
	}
	
	return details, nil
}

// ValidateLicenseKey validates the format of a license key
func ValidateLicenseKey(key string) bool {
	// Check format: MDF-XXXX-XXXX-XXXX
	parts := strings.Split(key, "-")
	if len(parts) != 4 {
		return false
	}
	
	if parts[0] != "MDF" {
		return false
	}
	
	// Check each part is 4 characters
	for i := 1; i < 4; i++ {
		if len(parts[i]) != 4 {
			return false
		}
		// Check characters are valid (A-Z, 2-9, excluding confusing ones)
		for _, c := range parts[i] {
			if !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789", c) {
				return false
			}
		}
	}
	
	return true
}

// ActivateLicense activates a license key by binding it to a device
func (cs *CryptoService) ActivateLicense(licenseKey string, deviceID string, plan LicensePlan) (*License, error) {
	// Validate license key format
	if !ValidateLicenseKey(licenseKey) {
		return nil, fmt.Errorf("invalid license key format")
	}
	
	// Validate device ID
	if deviceID == "" {
		return nil, fmt.Errorf("device ID is required")
	}
	
	// Generate license with device binding
	return cs.GenerateLicenseWithDevice(licenseKey, plan, deviceID)
}
