package valueobject

import (
	"testing"
)

func TestLicenseToEmail(t *testing.T) {
	license := &License{
		LicenseKey: "MDF-ABCD-EFGH-JKLM",
	}

	expected := "abcd-efgh-jklm@mdfriday.com"
	if got := license.ToEmail(); got != expected {
		t.Errorf("ToEmail() = %v, want %v", got, expected)
	}
}

func TestLicenseToPassword(t *testing.T) {
	license := &License{
		LicenseKey: "MDF-ABCD-EFGH-JKLM",
	}

	// base64("abcd-efgh-jklm") = "YWJjZC1lZmdoLWprbG0="
	expected := "YWJjZC1lZmdoLWprbG0="
	if got := license.ToPassword(); got != expected {
		t.Errorf("ToPassword() = %v, want %v", got, expected)
	}
}

func TestLicenseToUserDir(t *testing.T) {
	license := &License{
		LicenseKey: "MDF-ABCD-EFGH-JKLM",
	}

	userDir := license.ToUserDir()
	if len(userDir) != 16 {
		t.Errorf("ToUserDir() length = %v, want 16", len(userDir))
	}
}

func TestLicenseSetHash(t *testing.T) {
	license := &License{
		LicenseKey: "MDF-ABCD-EFGH-JKLM",
	}

	license.SetHash()
	if license.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(license.Hash) != 32 { // MD5 produces 32 hex characters
		t.Errorf("Hash length = %v, want 32", len(license.Hash))
	}
}

func TestLicenseIsExpired(t *testing.T) {
	tests := []struct {
		name       string
		expiryDate int64
		want       bool
	}{
		{"expired", 0, true},
		{"not expired", 9999999999999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &License{ExpiryDate: tt.expiryDate}
			if got := license.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLicenseIsValid(t *testing.T) {
	tests := []struct {
		name       string
		activated  bool
		expiryDate int64
		want       bool
	}{
		{"valid", true, 9999999999999, true},
		{"not activated", false, 9999999999999, false},
		{"expired", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &License{
				Activated:  tt.activated,
				ExpiryDate: tt.expiryDate,
			}
			if got := license.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLicenseCanAddDevice(t *testing.T) {
	tests := []struct {
		name           string
		maxDevices     int
		currentDevices int
		want           bool
	}{
		{"can add", 3, 2, true},
		{"at limit", 3, 3, false},
		{"unlimited", -1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &License{
				MaxDevices:     tt.maxDevices,
				CurrentDevices: tt.currentDevices,
			}
			if got := license.CanAddDevice(); got != tt.want {
				t.Errorf("CanAddDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLicenseCanAddIP(t *testing.T) {
	tests := []struct {
		name       string
		maxIPs     int
		currentIPs int
		want       bool
	}{
		{"can add", 3, 2, true},
		{"at limit", 3, 3, false},
		{"unlimited", -1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &License{
				MaxIPs:     tt.maxIPs,
				CurrentIPs: tt.currentIPs,
			}
			if got := license.CanAddIP(); got != tt.want {
				t.Errorf("CanAddIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlanFeatures(t *testing.T) {
	tests := []struct {
		plan         LicensePlan
		maxDevices   int
		syncEnabled  bool
		customDomain bool
	}{
		{PlanFree, 1, false, false},
		{PlanStarter, 3, true, false},
		{PlanCreator, 5, true, true},
		{PlanPro, 10, true, true},
		{PlanEnterprise, -1, true, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			features := GetPlanFeatures(tt.plan)
			if features.MaxDevices != tt.maxDevices {
				t.Errorf("MaxDevices = %v, want %v", features.MaxDevices, tt.maxDevices)
			}
			if features.SyncEnabled != tt.syncEnabled {
				t.Errorf("SyncEnabled = %v, want %v", features.SyncEnabled, tt.syncEnabled)
			}
			if features.CustomDomain != tt.customDomain {
				t.Errorf("CustomDomain = %v, want %v", features.CustomDomain, tt.customDomain)
			}
		})
	}
}

func TestLicenseGetFeatures(t *testing.T) {
	license := &License{
		Plan: PlanCreator,
	}

	features := license.GetFeatures()
	if features.MaxDevices != 5 {
		t.Errorf("MaxDevices = %v, want 5", features.MaxDevices)
	}
	if !features.CustomDomain {
		t.Error("CustomDomain should be true for Creator plan")
	}
}

