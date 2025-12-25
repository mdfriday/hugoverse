package valueobject

import (
	"testing"
)

func TestLicenseDeviceSetHash(t *testing.T) {
	device := &LicenseDevice{
		License:  "MDF-ABCD-EFGH-JKLM",
		DeviceID: "device-12345678-abcd",
	}

	device.SetHash()
	if device.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(device.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(device.Hash))
	}
}

func TestLicenseDeviceSetSlug(t *testing.T) {
	tests := []struct {
		name     string
		license  string
		deviceID string
		wantSlug string
	}{
		{
			name:     "normal device ID",
			license:  "MDF-ABCD-EFGH-JKLM",
			deviceID: "device-12345678-abcd",
			wantSlug: "MDF-ABCD-EFGH-JKLM:device-1",
		},
		{
			name:     "short device ID",
			license:  "MDF-ABCD-EFGH-JKLM",
			deviceID: "abc",
			wantSlug: "MDF-ABCD-EFGH-JKLM:abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &LicenseDevice{
				License:  tt.license,
				DeviceID: tt.deviceID,
			}
			device.SetSlug("")
			if device.Slug != tt.wantSlug {
				t.Errorf("SetSlug() = %v, want %v", device.Slug, tt.wantSlug)
			}
		})
	}
}

func TestLicenseDeviceString(t *testing.T) {
	device := &LicenseDevice{
		DeviceID:   "device-12345678-abcd",
		DeviceName: "MacBook Pro",
	}

	expected := "device-1 - MacBook Pro"
	if got := device.String(); got != expected {
		t.Errorf("String() = %v, want %v", got, expected)
	}
}

func TestLicenseDeviceIndexContent(t *testing.T) {
	device := &LicenseDevice{}
	if !device.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

