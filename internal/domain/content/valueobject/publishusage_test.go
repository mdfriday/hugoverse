package valueobject

import (
	"testing"
)

func TestPublishUsageSetHash(t *testing.T) {
	usage := &PublishUsage{
		License:    "MDF-ABCD-EFGH-JKLM",
		RecordedAt: 1234567890,
	}

	usage.SetHash()
	if usage.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(usage.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(usage.Hash))
	}
}

func TestPublishUsageSetSlug(t *testing.T) {
	usage := &PublishUsage{
		License:    "MDF-ABCD-EFGH-JKLM",
		RecordedAt: 1234567890,
	}

	usage.SetSlug("")
	expected := "MDF-ABCD-EFGH-JKLM:1234567890"
	if usage.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", usage.Slug, expected)
	}
}

func TestPublishUsageStoragePercentage(t *testing.T) {
	tests := []struct {
		name         string
		storageBytes int64
		quotaBytes   int64
		want         float64
	}{
		{"50%", 500, 1000, 50.0},
		{"100%", 1000, 1000, 100.0},
		{"0 quota", 500, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &PublishUsage{
				StorageBytes: tt.storageBytes,
				QuotaBytes:   tt.quotaBytes,
			}
			if got := usage.StoragePercentage(); got != tt.want {
				t.Errorf("StoragePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishUsageIsStorageOverQuota(t *testing.T) {
	tests := []struct {
		name         string
		storageBytes int64
		quotaBytes   int64
		want         bool
	}{
		{"under quota", 500, 1000, false},
		{"at quota", 1000, 1000, false},
		{"over quota", 1500, 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &PublishUsage{
				StorageBytes: tt.storageBytes,
				QuotaBytes:   tt.quotaBytes,
			}
			if got := usage.IsStorageOverQuota(); got != tt.want {
				t.Errorf("IsStorageOverQuota() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishUsageCanAddSite(t *testing.T) {
	tests := []struct {
		name      string
		siteCount int
		maxSites  int
		want      bool
	}{
		{"can add", 2, 3, true},
		{"at limit", 3, 3, false},
		{"unlimited", 100, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &PublishUsage{
				SiteCount: tt.siteCount,
				MaxSites:  tt.maxSites,
			}
			if got := usage.CanAddSite(); got != tt.want {
				t.Errorf("CanAddSite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishUsageIndexContent(t *testing.T) {
	usage := &PublishUsage{}
	if !usage.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

