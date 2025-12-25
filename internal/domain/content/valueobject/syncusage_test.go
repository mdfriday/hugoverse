package valueobject

import (
	"testing"
)

func TestSyncUsageSetHash(t *testing.T) {
	usage := &SyncUsage{
		SyncAccount: "MDF-ABCD-EFGH-JKLM",
		RecordedAt:  1234567890,
	}

	usage.SetHash()
	if usage.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(usage.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(usage.Hash))
	}
}

func TestSyncUsageSetSlug(t *testing.T) {
	usage := &SyncUsage{
		SyncAccount: "MDF-ABCD-EFGH-JKLM",
		RecordedAt:  1234567890,
	}

	usage.SetSlug(nil)
	expected := "MDF-ABCD-EFGH-JKLM:1234567890"
	if usage.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", usage.Slug, expected)
	}
}

func TestSyncUsageUsagePercentage(t *testing.T) {
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
			usage := &SyncUsage{
				StorageBytes: tt.storageBytes,
				QuotaBytes:   tt.quotaBytes,
			}
			if got := usage.UsagePercentage(); got != tt.want {
				t.Errorf("UsagePercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncUsageIsOverQuota(t *testing.T) {
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
			usage := &SyncUsage{
				StorageBytes: tt.storageBytes,
				QuotaBytes:   tt.quotaBytes,
			}
			if got := usage.IsOverQuota(); got != tt.want {
				t.Errorf("IsOverQuota() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncUsageIndexContent(t *testing.T) {
	usage := &SyncUsage{}
	if !usage.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

