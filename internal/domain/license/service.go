package license

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Service provides license management functionality
type Service struct {
	cryptoService *CryptoService
	keysDir       string
	licensesDir   string
}

// NewService creates a new license service with default paths
func NewService(keysDir string) (*Service, error) {
	// Use default licenses directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	licensesDir := filepath.Join(homeDir, ".mdfriday", "licenses")
	
	return NewServiceWithPaths(keysDir, licensesDir)
}

// NewServiceWithPaths creates a new license service with custom paths
func NewServiceWithPaths(keysDir, licensesDir string) (*Service, error) {
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}
	
	if err := os.MkdirAll(licensesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create licenses directory: %w", err)
	}

	return &Service{
		keysDir:     keysDir,
		licensesDir: licensesDir,
	}, nil
}

// GenerateKeys generates and saves new cryptographic keys
func (s *Service) GenerateKeys() error {
	cryptoService, err := NewCryptoService()
	if err != nil {
		return err
	}

	s.cryptoService = cryptoService

	// Save ECDSA key pair
	ecdsaKeyPair, err := cryptoService.GetECDSAKeyPair()
	if err != nil {
		return err
	}

	if err := s.saveKeyPair("ecdsa", ecdsaKeyPair); err != nil {
		return err
	}

	// Save RSA key pair
	rsaKeyPair, err := cryptoService.GetRSAKeyPair()
	if err != nil {
		return err
	}

	if err := s.saveKeyPair("rsa", rsaKeyPair); err != nil {
		return err
	}

	// Save Basic KEK key pair
	basicKEKKeyPair, err := cryptoService.GetBasicKEKKeyPair()
	if err != nil {
		return err
	}

	if err := s.saveKeyPair("basic_kek", basicKEKKeyPair); err != nil {
		return err
	}

	// Save Premium KEK key pair
	premiumKEKKeyPair, err := cryptoService.GetPremiumKEKKeyPair()
	if err != nil {
		return err
	}

	if err := s.saveKeyPair("premium_kek", premiumKEKKeyPair); err != nil {
		return err
	}

	fmt.Println("✅ Keys generated successfully!")
	fmt.Printf("📁 Keys saved to: %s\n", s.keysDir)
	fmt.Println("📋 Generated files:")
	fmt.Println("   - ecdsa_private.pem")
	fmt.Println("   - ecdsa_public.pem")
	fmt.Println("   - rsa_private.pem")
	fmt.Println("   - rsa_public.pem")
	fmt.Println("   - basic_kek_private.pem")
	fmt.Println("   - basic_kek_public.pem")
	fmt.Println("   - premium_kek_private.pem")
	fmt.Println("   - premium_kek_public.pem")

	return nil
}

// LoadKeys loads existing cryptographic keys
func (s *Service) LoadKeys() error {
	ecdsaPrivPath := filepath.Join(s.keysDir, "ecdsa_private.pem")
	rsaPrivPath := filepath.Join(s.keysDir, "rsa_private.pem")
	basicKEKPrivPath := filepath.Join(s.keysDir, "basic_kek_private.pem")
	premiumKEKPrivPath := filepath.Join(s.keysDir, "premium_kek_private.pem")

	// Check if keys exist
	if _, err := os.Stat(ecdsaPrivPath); os.IsNotExist(err) {
		return fmt.Errorf("ECDSA private key not found at %s. Please run 'hugov license keygen' first", ecdsaPrivPath)
	}

	if _, err := os.Stat(rsaPrivPath); os.IsNotExist(err) {
		return fmt.Errorf("RSA private key not found at %s. Please run 'hugov license keygen' first", rsaPrivPath)
	}

	if _, err := os.Stat(basicKEKPrivPath); os.IsNotExist(err) {
		return fmt.Errorf("Basic KEK private key not found at %s. Please run 'hugov license keygen' first", basicKEKPrivPath)
	}

	if _, err := os.Stat(premiumKEKPrivPath); os.IsNotExist(err) {
		return fmt.Errorf("Premium KEK private key not found at %s. Please run 'hugov license keygen' first", premiumKEKPrivPath)
	}

	// Load keys
	ecdsaPrivKey, err := ioutil.ReadFile(ecdsaPrivPath)
	if err != nil {
		return fmt.Errorf("failed to read ECDSA private key: %w", err)
	}

	rsaPrivKey, err := ioutil.ReadFile(rsaPrivPath)
	if err != nil {
		return fmt.Errorf("failed to read RSA private key: %w", err)
	}

	basicKEKPrivKey, err := ioutil.ReadFile(basicKEKPrivPath)
	if err != nil {
		return fmt.Errorf("failed to read basic KEK private key: %w", err)
	}

	premiumKEKPrivKey, err := ioutil.ReadFile(premiumKEKPrivPath)
	if err != nil {
		return fmt.Errorf("failed to read premium KEK private key: %w", err)
	}

	cryptoService, err := LoadCryptoService(string(ecdsaPrivKey), string(rsaPrivKey), string(basicKEKPrivKey), string(premiumKEKPrivKey))
	if err != nil {
		return err
	}

	s.cryptoService = cryptoService
	return nil
}

// GenerateLicenses generates licenses for the given request
func (s *Service) GenerateLicenses(req *LicenseRequest, outputDir string) error {
	// If no output directory specified, use the service's licenses directory
	if outputDir == "" {
		outputDir = s.licensesDir
	}
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return err
		}
	}

	details, err := s.cryptoService.GenerateLicenseDetails(req)
	if err != nil {
		return err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Collect license keys for registry
	var licenseKeys []string
	
	// Save each license detail to a separate file
	for _, detail := range details {
		// Create filename based on license key
		filename := fmt.Sprintf("%s.json", detail.LicenseKey)
		outputPath := filepath.Join(outputDir, filename)
		
		// Save license detail to file
		detailBytes, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal license detail %s: %w", detail.LicenseKey, err)
		}

		if err := ioutil.WriteFile(outputPath, detailBytes, 0644); err != nil {
			return fmt.Errorf("failed to write license detail file %s: %w", outputPath, err)
		}
		
		licenseKeys = append(licenseKeys, detail.LicenseKey)
	}

	// Create and save registry file
	registry := &LicenseRegistry{
		GeneratedAt: time.Now().Format("2006-01-02"),
		Plan:        string(req.Plan),
		Count:       req.Count,
		LicenseKeys: licenseKeys,
	}
	
	registryPath := filepath.Join(outputDir, fmt.Sprintf("licenses_%s_%s.json", req.Plan, registry.GeneratedAt))
	registryBytes, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := ioutil.WriteFile(registryPath, registryBytes, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	fmt.Println("✅ Licenses generated successfully!")
	fmt.Printf("📋 Plan: %s\n", req.Plan)
	fmt.Printf("🔢 Count: %d\n", req.Count)
	fmt.Printf("📁 Files saved to: %s\n", outputDir)
	fmt.Println("📋 Generated license keys:")
	for _, key := range licenseKeys {
		fmt.Printf("   - %s\n", key)
	}
	fmt.Printf("📄 Registry saved to: %s\n", registryPath)

	return nil
}

// GetPublicKeys returns the public keys for frontend use
func (s *Service) GetPublicKeys() (*PublicKeys, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return nil, err
		}
	}

	return s.cryptoService.GetPublicKeys()
}

// VerifyLicense verifies a license file
func (s *Service) VerifyLicense(licensePath string) error {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return err
		}
	}

	// Read license file
	licenseBytes, err := ioutil.ReadFile(licensePath)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	var license License
	if err := json.Unmarshal(licenseBytes, &license); err != nil {
		return fmt.Errorf("failed to parse license: %w", err)
	}

	// Decode payload from base64
	payloadBytes, err := base64.StdEncoding.DecodeString(license.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	// Verify signature
	valid, err := s.cryptoService.VerifySignature(&payload, license.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid license signature")
	}

	fmt.Println("✅ License is valid!")
	fmt.Printf("🔑 License Key: %s\n", payload.LicenseKey)
	if payload.DeviceID != "" {
		fmt.Printf("📱 Device ID: %s\n", payload.DeviceID)
	} else {
		fmt.Println("📱 Device ID: Not activated (unbound license)")
	}
	fmt.Printf("📋 Plan: %s\n", payload.Plan)
	fmt.Printf("📅 Issued at: %s\n", payload.IssueAt.Format("2006-01-02 15:04:05"))
	if payload.Exp != nil {
		fmt.Printf("⏰ Expires at: %s\n", payload.Exp.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("⏰ Never expires (lifetime license)")
	}

	return nil
}

// ActivateLicense activates a license key with device binding
func (s *Service) ActivateLicense(req *ActivationRequest) (*ActivationResponse, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return &ActivationResponse{
				Success:  false,
				ErrorMsg: fmt.Sprintf("Failed to load keys: %v", err),
			}, nil
		}
	}

	// Load license detail from file
	detail, err := s.loadLicenseDetail(req.LicenseKey)
	if err != nil {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("License key not found: %v", err),
		}, nil
	}

	// Check if license is expired
	if s.isLicenseExpired(detail.ExpiryDate) {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: "License key has expired",
		}, nil
	}

	// Check if device is already activated
	for _, deviceID := range detail.DeviceIDs {
		if deviceID == req.DeviceID {
			// Device already activated, generate license
			license, err := s.generateActivatedLicense(detail, req.DeviceID)
			if err != nil {
				return &ActivationResponse{
					Success:  false,
					ErrorMsg: fmt.Sprintf("Failed to generate license: %v", err),
				}, nil
			}

			publicKeys, err := s.GetPublicKeys()
			if err != nil {
				return &ActivationResponse{
					Success:  false,
					ErrorMsg: fmt.Sprintf("Failed to get public keys: %v", err),
				}, nil
			}

			return &ActivationResponse{
				Success:    true,
				License:    license,
				PublicKeys: publicKeys,
				Detail:     detail,
			}, nil
		}
	}

	// Check if we can add more devices
	if detail.CurrentActivations >= detail.MaxActivations {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Maximum activations reached (%d/%d)", detail.CurrentActivations, detail.MaxActivations),
		}, nil
	}

	// Add new device
	detail.DeviceIDs = append(detail.DeviceIDs, req.DeviceID)
	detail.CurrentActivations++

	// Save updated detail
	if err := s.saveLicenseDetail(detail); err != nil {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Failed to save activation: %v", err),
		}, nil
	}

	// Generate activated license
	license, err := s.generateActivatedLicense(detail, req.DeviceID)
	if err != nil {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Failed to generate license: %v", err),
		}, nil
	}

	// Get public keys for frontend
	publicKeys, err := s.GetPublicKeys()
	if err != nil {
		return &ActivationResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Failed to get public keys: %v", err),
		}, nil
	}

	return &ActivationResponse{
		Success:    true,
		License:    license,
		PublicKeys: publicKeys,
		Detail:     detail,
	}, nil
}

// loadLicenseDetail loads license detail from individual JSON file
func (s *Service) loadLicenseDetail(licenseKey string) (*LicenseDetail, error) {
	filename := fmt.Sprintf("%s.json", licenseKey)
	filePath := filepath.Join(s.licensesDir, filename)
	
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("license detail file not found for key: %s (expected at %s)", licenseKey, filePath)
	}
	
	// Read and parse the file
	detailBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read license detail file: %w", err)
	}
	
	var detail LicenseDetail
	if err := json.Unmarshal(detailBytes, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse license detail file: %w", err)
	}
	
	return &detail, nil
}

// saveLicenseDetail saves license detail to JSON file
func (s *Service) saveLicenseDetail(detail *LicenseDetail) error {
	filename := fmt.Sprintf("%s.json", detail.LicenseKey)
	filePath := filepath.Join(s.licensesDir, filename)
	
	// Ensure directory exists
	if err := os.MkdirAll(s.licensesDir, 0755); err != nil {
		return fmt.Errorf("failed to create licenses directory: %w", err)
	}
	
	// Marshal and save
	detailBytes, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal license detail: %w", err)
	}
	
	if err := ioutil.WriteFile(filePath, detailBytes, 0644); err != nil {
		return fmt.Errorf("failed to write license detail file: %w", err)
	}
	
	return nil
}

// isLicenseExpired checks if a license is expired based on expiry date string
func (s *Service) isLicenseExpired(expiryDate string) bool {
	if expiryDate == "9999-01-01" {
		return false // Lifetime license never expires
	}
	
	expiry, err := time.Parse("2006-01-02", expiryDate)
	if err != nil {
		return true // Invalid date format, consider expired
	}
	
	return expiry.Before(time.Now())
}

// generateActivatedLicense generates a license with device binding using existing resourceKey
func (s *Service) generateActivatedLicense(detail *LicenseDetail, deviceID string) (*License, error) {
	// Convert plan string to LicensePlan
	var plan LicensePlan
	switch detail.Plan {
	case PlanFree:
		plan = PlanFree
	case PlanYearly:
		plan = PlanYearly
	case PlanLifetime:
		plan = PlanLifetime
	default:
		return nil, fmt.Errorf("invalid plan: %s", detail.Plan)
	}
	
	// Create payload using existing resourceKeys from detail
	payload := &LicensePayload{
		LicenseKey:   detail.LicenseKey,
		DeviceID:     deviceID,
		Plan:         plan,
		ResourceKeys: detail.ResourceKeys,
		IssueAt:      time.Now(),
		Version:      1,
	}

	// Set expiration for yearly plans
	if plan == PlanYearly {
		exp := time.Now().AddDate(1, 0, 0) // 1 year from now
		payload.Exp = &exp
	}

	// Sign payload
	signature, err := s.cryptoService.SignPayload(payload)
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

// EncryptContent encrypts content and returns encrypted data and CEK
func (s *Service) EncryptContent(content []byte) ([]byte, []byte, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return nil, nil, err
		}
	}

	// Generate CEK
	cek, err := s.cryptoService.GenerateCEK()
	if err != nil {
		return nil, nil, err
	}

	// Encrypt content
	encryptedData, err := s.cryptoService.EncryptContent(cek, content)
	if err != nil {
		return nil, nil, err
	}

	return encryptedData, cek, nil
}

// EncryptContentWithLevel encrypts content with the specified access level
func (s *Service) EncryptContentWithLevel(content []byte, level ContentLevel) ([]byte, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return nil, err
		}
	}

	return s.cryptoService.EncryptContentWithLevel(content, level)
}

// DecryptContentWithLicense decrypts content using license payload
func (s *Service) DecryptContentWithLicense(payload *LicensePayload, encryptedData []byte) ([]byte, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return nil, err
		}
	}

	// Check if this is level-encrypted content (new format)
	if len(encryptedData) > 1 {
		// Try to decrypt as level-encrypted content first
		availableKEKs := make(map[ContentLevel]*rsa.PrivateKey)
		
		// Add basic KEK if license has basic access
		if _, hasBasic := payload.ResourceKeys["basic"]; hasBasic {
			availableKEKs[ContentLevelBasic] = s.cryptoService.basicKEKPrivateKey
		}
		
		// Add premium KEK if license has premium access
		if _, hasPremium := payload.ResourceKeys["premium"]; hasPremium {
			availableKEKs[ContentLevelPremium] = s.cryptoService.premiumKEKPrivateKey
		}
		
		// Try to decrypt with level format
		decryptedData, err := s.cryptoService.DecryptContentWithLevel(encryptedData, availableKEKs)
		if err == nil {
			return decryptedData, nil
		}
		// If level decryption fails, fall back to old format
	}

	// Fallback to old format (CEK-based) for backward compatibility
	// Try to get basic CEK first (most common case)
	var cek ContentEncryptionKey
	var err error
	
	if basicResourceKey, exists := payload.ResourceKeys["basic"]; exists {
		cek, err = s.cryptoService.UnwrapCEK(basicResourceKey)
		if err != nil {
			return nil, fmt.Errorf("failed to unwrap basic CEK: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no basic resource key found in license")
	}

	// Decrypt content
	decryptedData, err := s.cryptoService.DecryptContent(cek, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt content: %w", err)
	}

	return decryptedData, nil
}

// DecryptContentFromRequest decrypts content from API request
func (s *Service) DecryptContentFromRequest(req *DecryptRequest) (*DecryptResponse, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return &DecryptResponse{
				Success:  false,
				ErrorMsg: fmt.Sprintf("Failed to load keys: %v", err),
			}, nil
		}
	}

	// Decode license payload
	payloadBytes, err := base64.StdEncoding.DecodeString(req.License)
	if err != nil {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "Invalid license format",
		}, nil
	}

	var payload LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "Failed to parse license payload",
		}, nil
	}

	// Verify license signature
	isValid, err := s.cryptoService.VerifySignature(&payload, req.Signature)
	if err != nil {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "Failed to verify license signature",
		}, nil
	}

	if !isValid {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "Invalid license signature",
		}, nil
	}

	// Check license expiration
	if payload.Exp != nil && payload.Exp.Before(time.Now()) {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "License has expired",
		}, nil
	}

	// Decode encrypted content
	encryptedData, err := base64.StdEncoding.DecodeString(req.EncryptedContent)
	if err != nil {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: "Invalid encrypted content format",
		}, nil
	}

	// Decrypt content
	decryptedData, err := s.DecryptContentWithLicense(&payload, encryptedData)
	if err != nil {
		return &DecryptResponse{
			Success:  false,
			ErrorMsg: fmt.Sprintf("Failed to decrypt content: %v", err),
		}, nil
	}

	// Encode decrypted content as base64
	contentBase64 := base64.StdEncoding.EncodeToString(decryptedData)

	return &DecryptResponse{
		Success:     true,
		Content:     contentBase64,
		ContentType: "application/octet-stream", // Generic binary type
	}, nil
}

// EncryptContentWithLicenseKey encrypts content using the CEK from a specific license
func (s *Service) EncryptContentWithLicenseKey(licenseKey string, content []byte) ([]byte, error) {
	if s.cryptoService == nil {
		if err := s.LoadKeys(); err != nil {
			return nil, err
		}
	}

	// Load license detail
	detail, err := s.loadLicenseDetail(licenseKey)
	if err != nil {
		return nil, err
	}

	// Unwrap basic CEK from license
	basicResourceKey, exists := detail.ResourceKeys["basic"]
	if !exists {
		return nil, fmt.Errorf("no basic resource key found in license detail")
	}
	
	cek, err := s.cryptoService.UnwrapCEK(basicResourceKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap basic CEK: %w", err)
	}

	// Encrypt content
	encryptedData, err := s.cryptoService.EncryptContent(cek, content)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt content: %w", err)
	}

	return encryptedData, nil
}

// saveKeyPair saves a key pair to files
func (s *Service) saveKeyPair(keyType string, keyPair *KeyPair) error {
	privPath := filepath.Join(s.keysDir, fmt.Sprintf("%s_private.pem", keyType))
	pubPath := filepath.Join(s.keysDir, fmt.Sprintf("%s_public.pem", keyType))

	if err := ioutil.WriteFile(privPath, []byte(keyPair.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save %s private key: %w", keyType, err)
	}

	if err := ioutil.WriteFile(pubPath, []byte(keyPair.PublicKey), 0644); err != nil {
		return fmt.Errorf("failed to save %s public key: %w", keyType, err)
	}

	return nil
}
