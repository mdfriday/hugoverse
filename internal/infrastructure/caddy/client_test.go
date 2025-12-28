package caddy

import (
	"testing"
	"time"
)

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	config := &Config{
		AdminAPI:       "http://127.0.0.1:2019",
		DefaultBackend: "127.0.0.1:1314",
		CoreDomain:     "mdfriday.site",
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("Failed to create client")
	}

	if client.config.AdminAPI != "http://127.0.0.1:2019" {
		t.Errorf("Expected AdminAPI to be 'http://127.0.0.1:2019', got '%s'", client.config.AdminAPI)
	}
}

// TestSanitizeDomainForID 测试域名转换为 ID
func TestSanitizeDomainForID(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "example-com"},
		{"my-site.org", "my-site-org"},
		{"test123.io", "test123-io"},
		{"sub.domain.example.com", "sub-domain-example-com"},
	}

	for _, tt := range tests {
		result := sanitizeDomainForID(tt.domain)
		if result != tt.expected {
			t.Errorf("sanitizeDomainForID(%s) = %s; want %s", tt.domain, result, tt.expected)
		}
	}
}

// 以下是集成测试示例，需要实际的 Caddy 服务器才能运行

// TestStartServer 测试启动服务器（需要 Caddy 二进制文件）
func TestStartServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &Config{
		AdminAPI:       "http://127.0.0.1:2019",
		BinaryPath:     "caddy",
		DefaultBackend: "127.0.0.1:1314",
		CoreDomain:     "mdfriday.site",
	}

	client := NewClient(config)

	// 启动服务器
	err := client.StartServer()
	if err != nil {
		t.Logf("Failed to start server (this is expected if Caddy is not installed): %v", err)
		return
	}
	defer client.Stop()

	// 等待服务器完全启动
	time.Sleep(2 * time.Second)

	// 测试连接
	if err := client.Ping(); err != nil {
		t.Errorf("Failed to ping Caddy: %v", err)
	}
}

// TestAddStaticSite 测试添加静态站点
func TestAddStaticSite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &Config{
		AdminAPI: "http://127.0.0.1:2019",
	}

	client := NewClient(config)

	// 假设 Caddy 已经在运行
	err := client.AddStaticSite("example.com", "/web/sites/example-com")
	if err != nil {
		t.Logf("Failed to add static site (Caddy may not be running): %v", err)
	}
}

// TestGetCertificateStatus 测试获取证书状态
func TestGetCertificateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &Config{
		AdminAPI: "http://127.0.0.1:2019",
	}

	client := NewClient(config)

	// 假设 Caddy 已经在运行
	certInfo, err := client.GetCertificateStatus("example.com")
	if err != nil {
		t.Logf("Failed to get certificate status (Caddy may not be running): %v", err)
		return
	}

	t.Logf("Certificate status for example.com: %s", certInfo.Status)
}

