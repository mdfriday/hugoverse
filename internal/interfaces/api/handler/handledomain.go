package handler

import (
	"fmt"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mdfriday/hugoverse/internal/application"
	apiFrom "github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
)

// ========== SubDomain 管理 API ==========

// GetSubDomainHandler 获取当前 subdomain 信息
// GET /api/license/subdomain?key=xxx
func (s *Handler) GetSubDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 PublishEnabled 权限
	if !license.GetFeatures().PublishEnabled {
		s.jsonError(res, "Publish feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 获取 SubDomain 记录
	sd, err := s.contentApp.GetSubDomainByKey(pd.SubDomain)
	if err != nil {
		s.jsonError(res, "Subdomain not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
		"subdomain":   sd.Sub,
		"full_domain": fmt.Sprintf("%s.%s", sd.Sub, s.adminApp.Domain()),
		"folder":      pd.Folder,
		"created_at":  sd.Timestamp,
	})
}

// CheckSubDomainHandler 检查 subdomain 是否可用
// POST /api/license/subdomain/check
func (s *Handler) CheckSubDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	subdomain := req.PostForm.Get("subdomain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if subdomain == "" {
		s.jsonError(res, "Subdomain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 转换为小写
	subdomain = strings.ToLower(subdomain)

	// 验证格式
	if err := validateSubdomainFormat(subdomain); err != nil {
		s.jsonResponse(res, map[string]interface{}{
			"subdomain": subdomain,
			"available": false,
			"reason":    "invalid_format",
			"message":   err.Error(),
		})
		return
	}

	// 检查是否为保留字段
	if isReservedSubdomain(subdomain) {
		s.jsonResponse(res, map[string]interface{}{
			"subdomain": subdomain,
			"available": false,
			"reason":    "reserved",
			"message":   fmt.Sprintf("Subdomain '%s' is reserved", subdomain),
		})
		return
	}

	// 检查是否已被占用
	existingSd, err := s.contentApp.GetSubDomainByKey(subdomain)
	if err == nil && existingSd != nil {
		s.jsonResponse(res, map[string]interface{}{
			"subdomain": subdomain,
			"available": false,
			"reason":    "taken",
			"message":   "Subdomain is already taken",
		})
		return
	}

	// 可用
	s.jsonResponse(res, map[string]interface{}{
		"subdomain": subdomain,
		"available": true,
		"message":   "Subdomain is available",
	})
}

// UpdateSubDomainHandler 修改用户的 subdomain
// POST /api/license/subdomain/update
func (s *Handler) UpdateSubDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	newSubdomain := req.PostForm.Get("new_subdomain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if newSubdomain == "" {
		s.jsonError(res, "New subdomain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 PublishEnabled 权限
	if !license.GetFeatures().PublishEnabled {
		s.jsonError(res, "Publish feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 转换为小写
	newSubdomain = strings.ToLower(newSubdomain)

	// 验证新 subdomain 格式
	if err := validateSubdomainFormat(newSubdomain); err != nil {
		s.jsonError(res, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查是否为保留字段
	if isReservedSubdomain(newSubdomain) {
		s.jsonError(res, fmt.Sprintf("Subdomain '%s' is reserved", newSubdomain), http.StatusBadRequest)
		return
	}

	// 检查新 subdomain 是否已被占用
	existingSd, err := s.contentApp.GetSubDomainByKey(newSubdomain)
	if err == nil && existingSd != nil {
		s.jsonError(res, fmt.Sprintf("Subdomain '%s' is already taken", newSubdomain), http.StatusConflict)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 获取旧的 SubDomain 记录
	oldSd, err := s.contentApp.GetSubDomainByKey(pd.SubDomain)
	if err != nil {
		s.jsonError(res, "Current subdomain not found", http.StatusNotFound)
		return
	}

	oldSubdomain := oldSd.Sub
	oldFullDomain := fmt.Sprintf("%s.%s", oldSubdomain, s.adminApp.RootDomain())
	newFullDomain := fmt.Sprintf("%s.%s", newSubdomain, s.adminApp.RootDomain())

	// sitePath 保持不变
	sitePath := filepath.Join(application.PreviewDir(), pd.Folder, application.SubDomainFolder())

	// 1. 移除旧的 Caddy route
	if err := s.caddyClient.RemoveStaticSite(oldFullDomain); err != nil {
		s.log.Errorf("Failed to remove old Caddy route: %v", err)
		// 继续执行，不阻塞
	}

	// 2. 创建 SubDomain 记录
	if err := s.contentApp.DeleteSubDomain(oldSd); err != nil {
		s.log.Errorf("Failed to delete old subdomain record: %v", err)
		s.jsonError(res, "Failed to update subdomain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := timestamp.CurrentTimeMillis()
	newSd := &contentVO.SubDomain{
		License: license.LicenseKey,
		Sub:     newSubdomain,
		Item: contentVO.Item{
			Timestamp: now,
			Updated:   now,
			Namespace: "SubDomain",
		},
	}

	// 创建新的 SubDomain 记录
	if _, err := s.contentApp.CreateSubDomain(newSd); err != nil {
		s.log.Errorf("Failed to create new subdomain record: %v", err)
		// 回滚：恢复旧的 Caddy route
		if err := s.caddyClient.AddStaticSite(oldFullDomain, sitePath); err != nil {
			s.log.Errorf("Failed to rollback old Caddy route: %v", err)
		}
		s.jsonError(res, "Failed to update subdomain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. 更新 PublishDomain 记录
	pd.SubDomain = newSubdomain
	pd.Updated = now
	if err := s.contentApp.UpdatePublishDomain(pd); err != nil {
		s.log.Errorf("Failed to update publish domain: %v", err)
		s.jsonError(res, "Failed to update publish domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. 添加新的 Caddy route
	if err := s.caddyClient.AddStaticSite(newFullDomain, sitePath); err != nil {
		s.log.Errorf("Failed to add new Caddy route: %v", err)
		s.jsonError(res, "Failed to add Caddy route: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
		"old_subdomain": oldSubdomain,
		"new_subdomain": newSubdomain,
		"full_domain":   newFullDomain,
		"message":       "Subdomain updated successfully",
	})
}

// ========== 自定义域名管理 API ==========

// CheckDomainHandler 检查域名 DNS 配置（前置检测）
// POST /api/license/domain/check
// 注意：此 API 只检查 DNS，不检查 HTTPS 状态
// HTTPS 状态请使用 /api/license/domain/https-status
func (s *Handler) CheckDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	domain := req.PostForm.Get("domain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if domain == "" {
		s.jsonError(res, "Domain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 CustomDomain 权限
	if !license.GetFeatures().CustomDomain {
		s.jsonError(res, "Custom domain feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 调用 DNS 检查
	result, err := s.caddyClient.CheckDomainReadiness(domain)
	if err != nil {
		s.log.Errorf("Failed to check domain readiness: %v", err)
		s.jsonError(res, "Failed to check domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"domain":       domain,
		"dns_valid":    result.DNSValid,
		"resolved_ips": result.ResolvedIPs,
		"ready":        result.Ready,
	}

	if result.Ready {
		response["message"] = "Domain DNS is configured correctly. You can proceed to add this domain."
	} else if result.Error != "" {
		response["error"] = result.Error
	}

	s.jsonResponse(res, response)
}

// AddDomainHandler 添加自定义域名
// POST /api/license/domain/add
func (s *Handler) AddDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	domain := req.PostForm.Get("domain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if domain == "" {
		s.jsonError(res, "Domain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 CustomDomain 权限
	if !license.GetFeatures().CustomDomain {
		s.jsonError(res, "Custom domain feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 检查用户是否已有自定义域名
	if pd.CusDomain != "" {
		s.jsonError(res, fmt.Sprintf("You already have a custom domain: %s. Please remove it first.", pd.CusDomain), http.StatusConflict)
		return
	}

	// 检查域名是否已被其他用户使用
	// TODO: 实现全局域名唯一性检查（需要遍历所有 PublishDomain 或建立索引）

	// ✅ 只进行 DNS 检测（可选，用于快速反馈）
	dnsResult, err := s.caddyClient.CheckDomainReadiness(domain)
	if err != nil {
		s.log.Errorf("Failed to check domain DNS: %v", err)
		// 不阻塞，继续执行
	} else if !dnsResult.DNSValid {
		// DNS 未配置，返回友好提示
		serverIP := s.adminApp.ServerIP()
		if serverIP == "" {
			serverIP = "your server IP"
		}
		s.jsonError(res, fmt.Sprintf(
			"Domain DNS is not configured correctly. Please point your domain to %s and wait a few minutes for DNS propagation.",
			serverIP,
		), http.StatusBadRequest)
		return
	}

	// sitePath: 自定义域名内容目录
	sitePath := filepath.Join(application.PreviewDir(), s.db.UserDir(), application.CustomDomainFolder())

	// ✅ 添加域名（跳过重复检测）
	if err := s.caddyClient.AddCustomDomain(domain, sitePath, true); err != nil {
		s.log.Errorf("Failed to add custom domain to Caddy: %v", err)
		s.jsonError(res, "Failed to add custom domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 更新 PublishDomain 记录
	now := timestamp.CurrentTimeMillis()
	pd.CusDomain = domain
	pd.Updated = now
	if err := s.contentApp.UpdatePublishDomain(pd); err != nil {
		s.log.Errorf("Failed to update publish domain: %v", err)
		// 回滚：移除 Caddy 配置
		s.caddyClient.RemoveCustomDomain(domain)
		s.jsonError(res, "Failed to update publish domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ✅ 返回 pending 状态，引导用户使用 https-status API
	s.jsonResponse(res, map[string]interface{}{
		"domain":  domain,
		"status":  "cert_pending",
		"message": "Custom domain added successfully. HTTPS certificate is being issued by Let's Encrypt (usually 1-2 minutes). Use /api/license/domain/https-status to check certificate status.",
	})
}

// RemoveDomainHandler 移除自定义域名
// POST /api/license/domain/remove
func (s *Handler) RemoveDomainHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	domain := req.PostForm.Get("domain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if domain == "" {
		s.jsonError(res, "Domain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 验证域名属于该用户
	if pd.CusDomain != domain {
		s.jsonError(res, "Domain does not belong to this license", http.StatusForbidden)
		return
	}

	// 调用 Caddy 移除自定义域名（自动处理 route 和 TLS policy）
	if err := s.caddyClient.RemoveCustomDomain(domain); err != nil {
		s.log.Errorf("Failed to remove custom domain from Caddy: %v", err)
		// 不阻塞，继续更新数据库
	}

	// 更新 PublishDomain 记录
	now := timestamp.CurrentTimeMillis()
	pd.CusDomain = ""
	pd.Updated = now
	if err := s.contentApp.UpdatePublishDomain(pd); err != nil {
		s.log.Errorf("Failed to update publish domain: %v", err)
		s.jsonError(res, "Failed to update publish domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
		"domain":  domain,
		"message": "Custom domain removed successfully",
	})
}

// GetDomainsHandler 获取用户所有域名配置
// GET /api/license/domains?key=xxx
func (s *Handler) GetDomainsHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 PublishEnabled 权限
	if !license.GetFeatures().PublishEnabled {
		s.jsonError(res, "Publish feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 获取 SubDomain 记录
	sd, err := s.contentApp.GetSubDomainByKey(pd.SubDomain)
	if err != nil {
		s.jsonError(res, "Subdomain not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
		"subdomain":   sd.Sub,
		"full_domain": fmt.Sprintf("%s.%s", sd.Sub, s.adminApp.Domain()),
		"cus_domain":  pd.CusDomain,
		"folder":      pd.Folder,
		"created_at":  sd.Timestamp,
	})
}

// ========== 辅助函数 ==========

// validateSubdomainFormat 验证 subdomain 格式
// 用户修改的 subdomain 必须 >= 4 字符
func validateSubdomainFormat(subdomain string) error {
	// 最小长度 4
	if len(subdomain) < 4 {
		return fmt.Errorf("Subdomain must be at least 4 characters long")
	}

	// 最大长度 32
	if len(subdomain) > 32 {
		return fmt.Errorf("Subdomain must be at most 32 characters long")
	}

	// 只允许小写字母、数字、连字符
	// 不能以连字符开头或结尾
	// 正则: ^[a-z0-9][a-z0-9-]{2,30}[a-z0-9]$
	pattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,30}[a-z0-9]$`)
	if !pattern.MatchString(subdomain) {
		return fmt.Errorf("Subdomain can only contain lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen")
	}

	return nil
}

// isReservedSubdomain 检查是否为保留的 subdomain
func isReservedSubdomain(subdomain string) bool {
	reserved := map[string]bool{
		"www":          true,
		"api":          true,
		"admin":        true,
		"cdb":          true,
		"mail":         true,
		"ftp":          true,
		"smtp":         true,
		"pop":          true,
		"imap":         true,
		"ns1":          true,
		"ns2":          true,
		"ns3":          true,
		"mx":           true,
		"mx1":          true,
		"mx2":          true,
		"webmail":      true,
		"cpanel":       true,
		"whm":          true,
		"autodiscover": true,
		"autoconfig":   true,
		"test":         true,
		"dev":          true,
		"staging":      true,
		"prod":         true,
		"beta":         true,
		"alpha":        true,
		"demo":         true,
		"preview":      true,
		"support":      true,
		"help":         true,
		"docs":         true,
		"blog":         true,
		"shop":         true,
		"store":        true,
		"app":          true,
		"apps":         true,
		"mobile":       true,
		"status":       true,
		"cdn":          true,
		"static":       true,
		"assets":       true,
		"img":          true,
		"images":       true,
		"video":        true,
		"videos":       true,
		"media":        true,
		"download":     true,
		"downloads":    true,
		"upload":       true,
		"uploads":      true,
		"file":         true,
		"files":        true,
		"secure":       true,
		"ssl":          true,
		"vpn":          true,
		"proxy":        true,
		"git":          true,
		"svn":          true,
		"repo":         true,
		"jenkins":      true,
		"ci":           true,
		"build":        true,
		"deploy":       true,
		"monitor":      true,
		"log":          true,
		"logs":         true,
		"analytics":    true,
		"stats":        true,
		"db":           true,
		"database":     true,
		"mysql":        true,
		"postgres":     true,
		"redis":        true,
		"mongo":        true,
		"elastic":      true,
		"kafka":        true,
		"rabbit":       true,
		"mdfriday":     true,
	}

	return reserved[subdomain]
}

// DomainSSLStatusHandler 查询自定义域名 HTTPS 状态
// POST /api/license/domain/https-status
func (s *Handler) DomainSSLStatusHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory)
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	domain := req.PostForm.Get("domain")

	if licenseKey == "" {
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if domain == "" {
		s.jsonError(res, "Domain is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}
	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 检查 CustomDomain 权限
	if !license.GetFeatures().CustomDomain {
		s.jsonError(res, "Custom domain feature not enabled", http.StatusForbidden)
		return
	}

	// 获取 PublishDomain 记录
	pd, err := s.contentApp.GetPublishDomainByKey(license.LicenseKey)
	if err != nil {
		s.jsonError(res, "Publish domain not found", http.StatusNotFound)
		return
	}

	// 验证域名属于该用户
	if pd.CusDomain != domain {
		s.jsonError(res, "Domain does not belong to this license", http.StatusForbidden)
		return
	}

	// 1. 先检查 DNS（快速反馈）
	dnsResult, err := s.caddyClient.CheckDomainReadiness(domain)
	if err != nil {
		s.log.Errorf("Failed to check DNS: %v", err)
		s.jsonError(res, "Failed to check domain DNS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 如果 DNS 未配置，直接返回
	if !dnsResult.DNSValid {
		s.jsonResponse(res, map[string]interface{}{
			"domain":       domain,
			"status":       "dns_pending",
			"tls_ready":    false,
			"dns_valid":    false,
			"resolved_ips": dnsResult.ResolvedIPs,
			"error":        dnsResult.Error,
			"message":      "DNS is not configured correctly. Please point your domain to " + s.adminApp.ServerIP(),
		})
		return
	}

	// 2. 执行 TLS 检测（主判据）
	tlsResult := s.caddyClient.GetChecker().CheckTLS(domain)

	response := map[string]interface{}{
		"domain":       domain,
		"status":       tlsResult.TLSStatus,
		"tls_ready":    tlsResult.TLSReady,
		"dns_valid":    dnsResult.DNSValid,
		"resolved_ips": dnsResult.ResolvedIPs,
	}

	// 根据状态添加消息
	switch tlsResult.TLSStatus {
	case "active":
		response["message"] = "HTTPS is fully operational"
		if tlsResult.CertInfo != nil {
			response["certificate"] = map[string]interface{}{
				"status":      tlsResult.CertInfo.Status,
				"issuer":      tlsResult.CertInfo.Issuer,
				"subject":     tlsResult.CertInfo.Subject,
				"not_before":  tlsResult.CertInfo.NotBefore,
				"not_after":   tlsResult.CertInfo.NotAfter,
				"dns_names":   tlsResult.CertInfo.DNSNames,
				"is_wildcard": tlsResult.CertInfo.IsWildcard,
			}
		}

	case "cert_pending":
		response["message"] = "HTTPS certificate is being issued by Let's Encrypt. This usually takes 1-2 minutes."
		response["estimated_time"] = "1-2 minutes"
		response["troubleshooting"] = "If this persists for more than 5 minutes, please check your DNS configuration."

	case "cert_error":
		response["error"] = tlsResult.Error
		response["troubleshooting"] = "Please check DNS configuration and Caddy logs for details."

	default:
		response["status"] = "unknown"
		response["error"] = "Unknown TLS status"
	}

	s.jsonResponse(res, response)
}
