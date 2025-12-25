package valueobject

import (
	"testing"
)

func TestPublishSiteSetHash(t *testing.T) {
	site := &PublishSite{
		License: "MDF-ABCD-EFGH-JKLM",
		Name:    "my-blog",
	}

	site.SetHash()
	if site.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(site.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(site.Hash))
	}
}

func TestPublishSiteSetSlug(t *testing.T) {
	site := &PublishSite{
		License: "MDF-ABCD-EFGH-JKLM",
		Name:    "my-blog",
	}

	site.SetSlug("")
	expected := "MDF-ABCD-EFGH-JKLM:my-blog"
	if site.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", site.Slug, expected)
	}
}

func TestPublishSiteString(t *testing.T) {
	site := &PublishSite{
		Name: "my-blog",
	}

	if got := site.String(); got != "my-blog" {
		t.Errorf("String() = %v, want my-blog", got)
	}
}

func TestPublishSiteIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active", "active", true},
		{"pending", "pending", false},
		{"deleted", "deleted", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := &PublishSite{Status: tt.status}
			if got := site.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishSiteAbsAssetPath(t *testing.T) {
	tests := []struct {
		name      string
		asset     string
		uploadDir string
		wantErr   bool
	}{
		{"valid", "uploads/site.zip", "/data", false},
		{"empty asset", "", "/data", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := &PublishSite{Asset: tt.asset}
			_, err := site.AbsAssetPath(tt.uploadDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("AbsAssetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPublishSiteIndexContent(t *testing.T) {
	site := &PublishSite{}
	if !site.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

func TestPublishSiteDeploy(t *testing.T) {
	site := &PublishSite{}
	if !site.Deploy() {
		t.Error("Deploy() should return true")
	}
}

