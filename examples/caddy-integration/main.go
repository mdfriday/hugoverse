// Package caddy 提供了 Caddy 服务器的集成示例
//
// 使用场景：
// 1. 用户激活 License 后，自动创建 Sync 数据库和静态站点
// 2. 用户发布站点时，动态添加自定义域名
// 3. 查询用户站点的 SSL 证书状态

package main

import (
	"fmt"
	"log"

	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
)

// PublishService 发布服务示例
type PublishService struct {
	caddyClient *caddy.Client
	sitesRoot   string // 站点根目录，如 /web/sites
}

// NewPublishService 创建发布服务
func NewPublishService(caddyClient *caddy.Client, sitesRoot string) *PublishService {
	return &PublishService{
		caddyClient: caddyClient,
		sitesRoot:   sitesRoot,
	}
}

// PublishSite 发布用户站点
// userID: 用户 ID
// domain: 自定义域名（可选）
// siteName: 站点名称
func (s *PublishService) PublishSite(userID, domain, siteName string) error {
	// 1. 生成站点路径
	sitePath := fmt.Sprintf("%s/%s/%s", s.sitesRoot, userID, siteName)

	// 2. 确定域名
	if domain == "" {
		// 如果没有自定义域名，使用默认子域名
		domain = fmt.Sprintf("%s.mdfriday.site", userID)
	}

	// 3. 添加到 Caddy
	log.Printf("Publishing site: %s -> %s", domain, sitePath)
	if err := s.caddyClient.AddStaticSite(domain, sitePath); err != nil {
		return fmt.Errorf("failed to publish site: %w", err)
	}

	log.Printf("✅ Site published: %s", domain)
	return nil
}

// UnpublishSite 下线站点
func (s *PublishService) UnpublishSite(domain string) error {
	log.Printf("Unpublishing site: %s", domain)
	if err := s.caddyClient.RemoveStaticSite(domain); err != nil {
		return fmt.Errorf("failed to unpublish site: %w", err)
	}

	log.Printf("✅ Site unpublished: %s", domain)
	return nil
}

// GetSiteSSLStatus 获取站点 SSL 状态
func (s *PublishService) GetSiteSSLStatus(domain string) (string, error) {
	certInfo, err := s.caddyClient.GetCertificateStatus(domain)
	if err != nil {
		return "", fmt.Errorf("failed to get SSL status: %w", err)
	}

	return certInfo.Status, nil
}

// 使用示例
func main() {
	// 1. 初始化 Caddy 客户端
	config := &caddy.Config{
		AdminAPI:       "http://127.0.0.1:2019",
		DefaultBackend: "127.0.0.1:1314",
		CoreDomain:     "mdfriday.site",
	}
	caddyClient := caddy.NewClient(config)

	// 2. 创建发布服务
	publishService := NewPublishService(caddyClient, "/web/sites")

	// 3. 发布站点示例

	// 场景 1: 用户使用默认子域名
	if err := publishService.PublishSite("user123", "", "myblog"); err != nil {
		log.Fatalf("Failed to publish site: %v", err)
	}
	// 结果: user123.mdfriday.site -> /web/sites/user123/myblog

	// 场景 2: 用户使用自定义域名
	if err := publishService.PublishSite("user456", "blog.example.com", "website"); err != nil {
		log.Fatalf("Failed to publish site: %v", err)
	}
	// 结果: blog.example.com -> /web/sites/user456/website

	// 4. 查询 SSL 状态
	status, err := publishService.GetSiteSSLStatus("blog.example.com")
	if err != nil {
		log.Printf("Failed to get SSL status: %v", err)
	} else {
		fmt.Printf("SSL Status for blog.example.com: %s\n", status)
	}

	// 5. 下线站点
	if err := publishService.UnpublishSite("user123.mdfriday.site"); err != nil {
		log.Printf("Failed to unpublish site: %v", err)
	}
}

