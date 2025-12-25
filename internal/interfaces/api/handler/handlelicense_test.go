package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// MockLicenseRepository implements LicenseRepository for testing
type MockLicenseRepository struct {
	licenses map[string]*contentVO.License
}

func NewMockLicenseRepository() *MockLicenseRepository {
	return &MockLicenseRepository{
		licenses: make(map[string]*contentVO.License),
	}
}

func (r *MockLicenseRepository) GetLicenseByKey(key string) (*contentVO.License, error) {
	if license, ok := r.licenses[key]; ok {
		return license, nil
	}
	return nil, fmt.Errorf("license not found: %s", key)
}

func (r *MockLicenseRepository) UpdateLicense(license *contentVO.License) error {
	r.licenses[license.LicenseKey] = license
	return nil
}

func (r *MockLicenseRepository) CreateLicense(license *contentVO.License) error {
	license.ID = len(r.licenses) + 1
	r.licenses[license.LicenseKey] = license
	return nil
}

// MockSyncManager implements sync manager behavior for testing
type MockSyncManager struct {
	devices   map[string][]contentVO.LicenseDevice
	ips       map[string][]contentVO.LicenseIP
	accounts  map[string]*contentVO.SyncAccount
	callCount map[string]int
}

func NewMockSyncManager() *MockSyncManager {
	return &MockSyncManager{
		devices:   make(map[string][]contentVO.LicenseDevice),
		ips:       make(map[string][]contentVO.LicenseIP),
		accounts:  make(map[string]*contentVO.SyncAccount),
		callCount: make(map[string]int),
	}
}

func (m *MockSyncManager) ValidateAndRecordAccess(licenseKey, deviceID, deviceName, deviceType, ipAddress string) error {
	m.callCount["ValidateAndRecordAccess"]++
	return nil
}

func (m *MockSyncManager) CreateSyncAccount(license *contentVO.License) (*contentVO.SyncAccount, error) {
	m.callCount["CreateSyncAccount"]++
	account := &contentVO.SyncAccount{
		License:    license.LicenseKey,
		Email:      license.ToEmail(),
		DBName:     "userdb-" + license.ToUserDir(),
		DBEndpoint: "http://localhost:5984/userdb-" + license.ToUserDir(),
		Status:     "active",
		CreatedAt:  time.Now().UnixMilli(),
	}
	m.accounts[license.LicenseKey] = account
	return account, nil
}

func (m *MockSyncManager) GetSyncAccount(licenseKey string) (*contentVO.SyncAccount, error) {
	if account, ok := m.accounts[licenseKey]; ok {
		return account, nil
	}
	return nil, nil
}

func (m *MockSyncManager) GetDevices(licenseKey string) ([]contentVO.LicenseDevice, error) {
	return m.devices[licenseKey], nil
}

func (m *MockSyncManager) GetIPs(licenseKey string) ([]contentVO.LicenseIP, error) {
	return m.ips[licenseKey], nil
}

func (m *MockSyncManager) UpdateUsage(licenseKey string) (*contentVO.SyncUsage, error) {
	return &contentVO.SyncUsage{
		SyncAccount:   licenseKey,
		DocumentCount: 10,
		StorageBytes:  1024 * 1024,
		QuotaBytes:    500 * 1024 * 1024,
		RecordedAt:    time.Now().UnixMilli(),
	}, nil
}

func (m *MockSyncManager) BlockDevice(licenseKey, deviceID string) error {
	return nil
}

func (m *MockSyncManager) BlockIP(licenseKey, ipAddress string) error {
	return nil
}

// MockPublishManager implements publish manager behavior for testing
type MockPublishManager struct {
	sites   map[string][]contentVO.PublishSite
	domains map[string][]contentVO.PublishDomain
}

func NewMockPublishManager() *MockPublishManager {
	return &MockPublishManager{
		sites:   make(map[string][]contentVO.PublishSite),
		domains: make(map[string][]contentVO.PublishDomain),
	}
}

func (m *MockPublishManager) GetSites(licenseKey string) ([]contentVO.PublishSite, error) {
	return m.sites[licenseKey], nil
}

func (m *MockPublishManager) GetUsage(licenseKey string) (*contentVO.PublishUsage, error) {
	return &contentVO.PublishUsage{
		License:      licenseKey,
		SiteCount:    len(m.sites[licenseKey]),
		StorageBytes: 1024 * 1024,
		MaxSites:     10,
		QuotaBytes:   5 * 1024 * 1024 * 1024,
		RecordedAt:   time.Now().UnixMilli(),
	}, nil
}

func (m *MockPublishManager) GetDomains(licenseKey string) ([]contentVO.PublishDomain, error) {
	return m.domains[licenseKey], nil
}

func TestDetectPlanFromKey(t *testing.T) {
	h := &LicenseAPIHandler{}

	tests := []struct {
		key      string
		expected contentVO.LicensePlan
	}{
		{"MDF-FREE-XXXX-YYYY", contentVO.PlanFree},
		{"MDF-STARTER-XXXX-YYYY", contentVO.PlanStarter},
		{"MDF-CREATOR-XXXX-YYYY", contentVO.PlanCreator},
		{"MDF-PRO-XXXX-YYYY", contentVO.PlanPro},
		{"MDF-ENT-XXXX-YYYY", contentVO.PlanEnterprise},
		{"MDF-ENTERPRISE-XXXX-YYYY", contentVO.PlanEnterprise},
		{"MDF-ABCD-EFGH-IJKL", contentVO.PlanStarter}, // Default
	}

	for _, tc := range tests {
		result := h.detectPlanFromKey(tc.key)
		if result != tc.expected {
			t.Errorf("detectPlanFromKey(%s) = %s, expected %s", tc.key, result, tc.expected)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	h := &LicenseAPIHandler{}

	tests := []struct {
		name           string
		xForwardedFor  string
		xRealIP        string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:           "X-Forwarded-For single",
			xForwardedFor:  "192.168.1.1",
			xRealIP:        "",
			remoteAddr:     "10.0.0.1:12345",
			expectedIP:     "192.168.1.1",
		},
		{
			name:           "X-Forwarded-For multiple",
			xForwardedFor:  "192.168.1.1, 10.0.0.2",
			xRealIP:        "",
			remoteAddr:     "10.0.0.1:12345",
			expectedIP:     "192.168.1.1",
		},
		{
			name:           "X-Real-IP",
			xForwardedFor:  "",
			xRealIP:        "192.168.1.2",
			remoteAddr:     "10.0.0.1:12345",
			expectedIP:     "192.168.1.2",
		},
		{
			name:           "RemoteAddr only",
			xForwardedFor:  "",
			xRealIP:        "",
			remoteAddr:     "192.168.1.3:12345",
			expectedIP:     "192.168.1.3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwardedFor)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			req.RemoteAddr = tc.remoteAddr

			result := h.getClientIP(req)
			if result != tc.expectedIP {
				t.Errorf("getClientIP() = %s, expected %s", result, tc.expectedIP)
			}
		})
	}
}

func TestGetLicenseHandler_NotFound(t *testing.T) {
	repo := NewMockLicenseRepository()
	h := &LicenseAPIHandler{repo: repo}

	req := httptest.NewRequest("GET", "/api/license/v2/info?key=NOT-EXIST", nil)
	w := httptest.NewRecorder()

	h.GetLicenseHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetLicenseHandler_MissingKey(t *testing.T) {
	repo := NewMockLicenseRepository()
	h := &LicenseAPIHandler{repo: repo}

	req := httptest.NewRequest("GET", "/api/license/v2/info", nil)
	w := httptest.NewRecorder()

	h.GetLicenseHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetLicenseHandler_Found(t *testing.T) {
	repo := NewMockLicenseRepository()
	license := &contentVO.License{
		LicenseKey:  "MDF-TEST-1234-5678",
		Plan:        contentVO.PlanStarter,
		Activated:   true,
		ExpiryDate:  time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
		MaxDevices:  3,
		MaxIPs:      3,
	}
	repo.licenses["MDF-TEST-1234-5678"] = license

	h := &LicenseAPIHandler{repo: repo}

	req := httptest.NewRequest("GET", "/api/license/v2/info?key=MDF-TEST-1234-5678", nil)
	w := httptest.NewRecorder()

	h.GetLicenseHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["license_key"] != "MDF-TEST-1234-5678" {
		t.Errorf("Expected license_key MDF-TEST-1234-5678, got %v", response["license_key"])
	}

	if response["plan"] != "starter" {
		t.Errorf("Expected plan starter, got %v", response["plan"])
	}
}

func TestActivateHandler_MissingLicenseKey(t *testing.T) {
	h := &LicenseAPIHandler{
		repo: NewMockLicenseRepository(),
	}

	body := `{"device_id": "device-123"}`
	req := httptest.NewRequest("POST", "/api/license/v2/activate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ActivateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestActivateHandler_MissingDeviceID(t *testing.T) {
	h := &LicenseAPIHandler{
		repo: NewMockLicenseRepository(),
	}

	body := `{"license_key": "MDF-TEST-1234-5678"}`
	req := httptest.NewRequest("POST", "/api/license/v2/activate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ActivateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestJSONResponse(t *testing.T) {
	h := &LicenseAPIHandler{}
	w := httptest.NewRecorder()

	h.jsonResponse(w, map[string]string{"key": "value"})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json")
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["key"] != "value" {
		t.Errorf("Expected key=value, got %v", response)
	}
}

func TestJSONError(t *testing.T) {
	h := &LicenseAPIHandler{}
	w := httptest.NewRecorder()

	h.jsonError(w, "Test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json")
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}

	if response["error"] != "Test error" {
		t.Errorf("Expected error='Test error', got %v", response["error"])
	}
}

