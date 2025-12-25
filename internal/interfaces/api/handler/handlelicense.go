package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// ========== License API Handlers ==========

// CreateLicenseHandler 创建 License (管理员接口)
// POST /api/license/create
func (h *Handler) CreateLicenseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		Plan       string `json:"plan"`
		ExpiryDays int    `json:"expiry_days"` // 有效期天数，默认 365
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.LicenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	// 检查 License 是否已存在
	existing, _ := h.contentApp.GetLicenseByKey(req.LicenseKey)
	if existing != nil {
		h.jsonError(w, "License already exists", http.StatusConflict)
		return
	}

	// 解析 Plan
	plan := contentVO.LicensePlan(req.Plan)
	if plan == "" {
		plan = contentVO.PlanStarter // 默认 Starter
	}

	// 计算过期时间
	expiryDays := req.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = 365 // 默认一年
	}

	now := time.Now()
	license := &contentVO.License{
		LicenseKey: req.LicenseKey,
		Plan:       plan,
		Activated:  false,
		IssueDate:  now.UnixMilli(),
		ExpiryDate: now.Add(time.Duration(expiryDays) * 24 * time.Hour).UnixMilli(),
		MaxDevices: contentVO.GetPlanFeatures(plan).MaxDevices,
		MaxIPs:     contentVO.GetPlanFeatures(plan).MaxIPs,
	}

	if err := h.contentApp.CreateLicense(license); err != nil {
		h.jsonError(w, "Failed to create license: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"success":     true,
		"message":     "License created successfully",
		"license_key": license.LicenseKey,
		"plan":        license.Plan,
		"issue_date":  license.IssueDate,
		"expires_at":  license.ExpiryDate,
		"features":    license.GetFeatures(),
	})
}

// ActivateLicenseHandler 激活 License
// POST /api/license/activate
func (h *Handler) ActivateLicenseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
		DeviceType string `json:"device_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.LicenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		h.jsonError(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	// 获取客户端 IP
	ipAddress := h.getClientIP(r)

	// 获取或创建 License
	license, err := h.contentApp.GetLicenseByKey(req.LicenseKey)
	if err != nil {
		// License 不存在，根据 LicenseKey 格式判断套餐
		plan := h.detectPlanFromKey(req.LicenseKey)
		features := contentVO.GetPlanFeatures(plan)

		license = &contentVO.License{
			LicenseKey:     req.LicenseKey,
			Plan:           plan,
			Activated:      true,
			ActivatedAt:    time.Now().UnixMilli(),
			IssueDate:      time.Now().UnixMilli(),
			ExpiryDate:     time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
			MaxDevices:     features.MaxDevices,
			MaxIPs:         features.MaxIPs,
			CurrentDevices: 0,
			CurrentIPs:     0,
		}
		if err := h.contentApp.CreateLicense(license); err != nil {
			h.jsonError(w, "Failed to create license: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 验证 License
	if !license.Activated {
		license.Activated = true
		license.ActivatedAt = time.Now().UnixMilli()
		h.contentApp.UpdateLicense(license)
	}

	if license.IsExpired() {
		h.jsonError(w, "License has expired", http.StatusForbidden)
		return
	}

	// 设置默认设备类型
	if req.DeviceType == "" {
		req.DeviceType = "desktop"
	}

	// 验证设备和 IP
	if h.syncManager != nil {
		if err := h.syncManager.ValidateAndRecordAccess(
			req.LicenseKey, req.DeviceID, req.DeviceName, req.DeviceType, ipAddress,
		); err != nil {
			h.jsonError(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	// 创建 Sync 账号 (如果支持)
	var syncInfo map[string]interface{}
	if h.syncManager != nil && license.GetFeatures().SyncEnabled {
		syncAccount, err := h.syncManager.CreateSyncAccount(license)
		if err != nil {
			h.log.Errorf("Failed to create sync account for license %s: %v", license.LicenseKey, err)
		} else if syncAccount != nil {
			syncInfo = map[string]interface{}{
				"email":       syncAccount.Email,
				"db_name":     syncAccount.DBName,
				"db_endpoint": syncAccount.DBEndpoint,
				"status":      syncAccount.Status,
			}
		}
	}

	// 返回响应
	response := map[string]interface{}{
		"success":     true,
		"license_key": license.LicenseKey,
		"plan":        license.Plan,
		"activated":   license.Activated,
		"expires_at":  license.ExpiryDate,
		"features":    license.GetFeatures(),
		"user": map[string]interface{}{
			"email":    license.ToEmail(),
			"user_dir": license.ToUserDir(),
		},
	}

	if syncInfo != nil {
		response["sync"] = syncInfo
	}

	h.jsonResponse(w, response)
}

// GetLicenseInfoHandler 获取 License 信息
// GET /api/license/info?key=xxx
func (h *Handler) GetLicenseInfoHandler(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("key")
	if licenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	license, err := h.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		h.jsonError(w, "License not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"license_key":     license.LicenseKey,
		"plan":            license.Plan,
		"activated":       license.Activated,
		"activated_at":    license.ActivatedAt,
		"issue_date":      license.IssueDate,
		"expires_at":      license.ExpiryDate,
		"is_expired":      license.IsExpired(),
		"is_valid":        license.IsValid(),
		"current_devices": license.CurrentDevices,
		"max_devices":     license.MaxDevices,
		"current_ips":     license.CurrentIPs,
		"max_ips":         license.MaxIPs,
		"features":        license.GetFeatures(),
	}

	h.jsonResponse(w, response)
}

// GetDevicesHandler 获取 License 的设备列表
// GET /api/license/devices?key=xxx
func (h *Handler) GetDevicesHandler(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("key")
	if licenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	if h.syncManager == nil {
		h.jsonError(w, "Sync manager not available", http.StatusServiceUnavailable)
		return
	}

	devices, err := h.syncManager.GetDevices(licenseKey)
	if err != nil {
		h.jsonError(w, "Failed to get devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetIPsHandler 获取 License 的 IP 列表
// GET /api/license/ips?key=xxx
func (h *Handler) GetIPsHandler(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("key")
	if licenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	if h.syncManager == nil {
		h.jsonError(w, "Sync manager not available", http.StatusServiceUnavailable)
		return
	}

	ips, err := h.syncManager.GetIPs(licenseKey)
	if err != nil {
		h.jsonError(w, "Failed to get IPs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"ips":   ips,
		"count": len(ips),
	})
}

// GetSyncInfoHandler 获取 Sync 信息
// GET /api/license/sync?key=xxx
func (h *Handler) GetSyncInfoHandler(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("key")
	if licenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	if h.syncManager == nil {
		h.jsonError(w, "Sync manager not available", http.StatusServiceUnavailable)
		return
	}

	account, err := h.syncManager.GetSyncAccount(licenseKey)
	if err != nil {
		h.jsonError(w, "Sync account not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"email":       account.Email,
		"db_name":     account.DBName,
		"db_endpoint": account.DBEndpoint,
		"status":      account.Status,
		"created_at":  account.CreatedAt,
	}

	// 尝试获取使用量
	usage, err := h.syncManager.UpdateUsage(licenseKey)
	if err == nil && usage != nil {
		response["usage"] = map[string]interface{}{
			"document_count": usage.DocumentCount,
			"storage_bytes":  usage.StorageBytes,
			"quota_bytes":    usage.QuotaBytes,
			"percentage":     usage.UsagePercentage(),
			"over_quota":     usage.IsOverQuota(),
		}
	}

	h.jsonResponse(w, response)
}

// GetPublishInfoHandler 获取 Publish 信息
// GET /api/license/publish?key=xxx
func (h *Handler) GetPublishInfoHandler(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("key")
	if licenseKey == "" {
		h.jsonError(w, "License key is required", http.StatusBadRequest)
		return
	}

	if h.publishManager == nil {
		h.jsonError(w, "Publish manager not available", http.StatusServiceUnavailable)
		return
	}

	// 获取站点列表
	sites, err := h.publishManager.GetSites(licenseKey)
	if err != nil {
		sites = []contentVO.PublishSite{}
	}

	// 获取使用量
	usage, _ := h.publishManager.GetUsage(licenseKey)

	// 获取域名列表
	domains, err := h.publishManager.GetDomains(licenseKey)
	if err != nil {
		domains = []contentVO.PublishDomain{}
	}

	response := map[string]interface{}{
		"sites":   sites,
		"domains": domains,
	}

	if usage != nil {
		response["usage"] = map[string]interface{}{
			"site_count":    usage.SiteCount,
			"max_sites":     usage.MaxSites,
			"storage_bytes": usage.StorageBytes,
			"quota_bytes":   usage.QuotaBytes,
			"storage_pct":   usage.StoragePercentage(),
			"over_quota":    usage.IsStorageOverQuota(),
			"can_add_site":  usage.CanAddSite(),
		}
	}

	h.jsonResponse(w, response)
}

// BlockDeviceHandler 封禁设备
// POST /api/license/device/block
func (h *Handler) BlockDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.syncManager == nil {
		h.jsonError(w, "Sync manager not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		DeviceID   string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.syncManager.BlockDevice(req.LicenseKey, req.DeviceID); err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "Device blocked successfully",
	})
}

// BlockIPHandler 封禁 IP
// POST /api/license/ip/block
func (h *Handler) BlockIPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.syncManager == nil {
		h.jsonError(w, "Sync manager not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		LicenseKey string `json:"license_key"`
		IPAddress  string `json:"ip_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.syncManager.BlockIP(req.LicenseKey, req.IPAddress); err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "IP blocked successfully",
	})
}

// ========== Helper Methods ==========

func (h *Handler) getClientIP(r *http.Request) string {
	// 优先检查 X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个 IP (可能有多个代理)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	// 检查 X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// 使用 RemoteAddr (去掉端口)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

func (h *Handler) detectPlanFromKey(key string) contentVO.LicensePlan {
	// 根据 License Key 前缀判断套餐类型
	// 例如: MDF-FREE-xxxx, MDF-STARTER-xxxx, MDF-CREATOR-xxxx, MDF-PRO-xxxx, MDF-ENT-xxxx
	upperKey := strings.ToUpper(key)
	switch {
	case strings.Contains(upperKey, "-FREE-"):
		return contentVO.PlanFree
	case strings.Contains(upperKey, "-STARTER-"):
		return contentVO.PlanStarter
	case strings.Contains(upperKey, "-CREATOR-"):
		return contentVO.PlanCreator
	case strings.Contains(upperKey, "-PRO-"):
		return contentVO.PlanPro
	case strings.Contains(upperKey, "-ENT-"), strings.Contains(upperKey, "-ENTERPRISE-"):
		return contentVO.PlanEnterprise
	default:
		// 默认 Starter
		return contentVO.PlanStarter
	}
}

func (h *Handler) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
