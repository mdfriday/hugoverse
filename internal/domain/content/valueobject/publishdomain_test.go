package valueobject

import (
	"strings"
	"testing"
)

func TestPublishDomainSetHash(t *testing.T) {
	domain := &PublishDomain{
		License: "MDF-ABCD-EFGH-JKLM",
		Domain:  "blog.example.com",
	}

	domain.SetHash()
	if domain.Hash == "" {
		t.Error("SetHash() should set a non-empty hash")
	}
	if len(domain.Hash) != 32 {
		t.Errorf("Hash length = %v, want 32", len(domain.Hash))
	}
}

func TestPublishDomainSetSlug(t *testing.T) {
	domain := &PublishDomain{
		License: "MDF-ABCD-EFGH-JKLM",
		Domain:  "blog.example.com",
	}

	domain.SetSlug("")
	expected := "MDF-ABCD-EFGH-JKLM:blog.example.com"
	if domain.Slug != expected {
		t.Errorf("SetSlug() = %v, want %v", domain.Slug, expected)
	}
}

func TestPublishDomainString(t *testing.T) {
	domain := &PublishDomain{
		Domain: "blog.example.com",
	}

	if got := domain.String(); got != "blog.example.com" {
		t.Errorf("String() = %v, want blog.example.com", got)
	}
}

func TestPublishDomainIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active", "active", true},
		{"inactive", "inactive", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := &PublishDomain{Status: tt.status}
			if got := domain.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublishDomainToCaddyConfig(t *testing.T) {
	domain := &PublishDomain{
		Domain:     "blog.example.com",
		TargetPath: "/data/publish/user123/sites/my-blog",
	}

	config := domain.ToCaddyConfig()

	if !strings.Contains(config, "blog.example.com") {
		t.Error("ToCaddyConfig() should contain the domain")
	}
	if !strings.Contains(config, "/data/publish/user123/sites/my-blog") {
		t.Error("ToCaddyConfig() should contain the target path")
	}
	if !strings.Contains(config, "file_server") {
		t.Error("ToCaddyConfig() should contain file_server directive")
	}
}

func TestPublishDomainIndexContent(t *testing.T) {
	domain := &PublishDomain{}
	if !domain.IndexContent() {
		t.Error("IndexContent() should return true")
	}
}

