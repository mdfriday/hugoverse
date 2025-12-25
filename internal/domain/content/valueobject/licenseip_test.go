package valueobject

import (
	"testing"
)

func TestLicenseIPSetHash(t *testing.T) {
	ip := &LicenseIP{
		License:   "MDF-ABCD-EFGH-JKLM",
		IPAddress: "192.168.1.100",
	}

	ip.SetHash()
	if ip.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(ip.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(ip.Hash))
	}
}

func TestLicenseIPSetSlug(t *testing.T) {
	ip := &LicenseIP{
		License:   "MDF-ABCD-EFGH-JKLM",
		IPAddress: "192.168.1.100",
	}

	ip.SetSlug("")
	expected := "MDF-ABCD-EFGH-JKLM:192.168.1.100"
	if ip.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", ip.Slug, expected)
	}
}

func TestLicenseIPString(t *testing.T) {
	tests := []struct {
		name      string
		ipAddress string
		country   string
		want      string
	}{
		{
			name:      "with country",
			ipAddress: "192.168.1.100",
			country:   "China",
			want:      "192.168.1.100 (China)",
		},
		{
			name:      "without country",
			ipAddress: "192.168.1.100",
			country:   "",
			want:      "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &LicenseIP{
				IPAddress: tt.ipAddress,
				Country:   tt.country,
			}
			if got := ip.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLicenseIPIndexContent(t *testing.T) {
	ip := &LicenseIP{}
	if !ip.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

