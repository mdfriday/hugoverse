package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mdfriday/hugoverse/internal/domain/license"
)

// LicenseHandler handles license-related API requests
type LicenseHandler struct {
	service *license.Service
}

// NewLicenseHandler creates a new license handler
func NewLicenseHandler() (*LicenseHandler, error) {
	// Get default directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	keysDir := filepath.Join(homeDir, ".mdfriday", "keys")
	licensesDir := filepath.Join(homeDir, ".mdfriday", "licenses")
	
	service, err := license.NewServiceWithPaths(keysDir, licensesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create license service: %w", err)
	}
	
	return &LicenseHandler{
		service: service,
	}, nil
}

// ActivateLicenseHandler handles license activation requests
func (h *LicenseHandler) ActivateLicenseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req license.ActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.LicenseKey == "" {
		http.Error(w, "License key is required", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	// Validate license key format
	if !license.ValidateLicenseKey(req.LicenseKey) {
		http.Error(w, "Invalid license key format", http.StatusBadRequest)
		return
	}

	// Activate license
	response, err := h.service.ActivateLicense(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	
	if !response.Success {
		w.WriteHeader(http.StatusBadRequest)
	}

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetPublicKeysHandler returns the public keys for frontend verification
func (h *LicenseHandler) GetPublicKeysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	publicKeys, err := h.service.GetPublicKeys()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get public keys: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(publicKeys); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ValidateLicenseKeyHandler validates a license key format without activation
func (h *LicenseHandler) ValidateLicenseKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		LicenseKey string `json:"licenseKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.LicenseKey == "" {
		http.Error(w, "License key is required", http.StatusBadRequest)
		return
	}

	// Validate format
	isValid := license.ValidateLicenseKey(req.LicenseKey)
	
	response := map[string]interface{}{
		"valid":      isValid,
		"licenseKey": req.LicenseKey,
	}

	if isValid {
		// Try to load license detail to get more info
		detail, err := h.loadLicenseDetailSafely(req.LicenseKey)
		if err == nil {
			response["plan"] = detail.Plan
			response["expiryDate"] = detail.ExpiryDate
			response["maxActivations"] = detail.MaxActivations
			response["currentActivations"] = detail.CurrentActivations
			response["expired"] = h.isLicenseExpired(detail.ExpiryDate)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Helper methods

func (h *LicenseHandler) loadLicenseDetailSafely(licenseKey string) (*license.LicenseDetail, error) {
	// Get default licenses directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	licensesDir := filepath.Join(homeDir, ".mdfriday", "licenses")
	
	filename := fmt.Sprintf("%s.json", licenseKey)
	filePath := filepath.Join(licensesDir, filename)
	
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("license detail not found")
	}
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open license file: %w", err)
	}
	defer file.Close()
	
	var detail license.LicenseDetail
	if err := json.NewDecoder(file).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %w", err)
	}
	
	return &detail, nil
}

func (h *LicenseHandler) isLicenseExpired(expiryDate string) bool {
	if expiryDate == "9999-01-01" {
		return false // Lifetime license never expires
	}
	
	expiry, err := time.Parse("2006-01-02", expiryDate)
	if err != nil {
		return true // Invalid date format, consider expired
	}
	
	return expiry.Before(time.Now())
}
