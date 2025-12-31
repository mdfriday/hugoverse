package handler

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mdfriday/hugoverse/internal/application"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	apiFrom "github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/pkg/timestamp"
	"net/http"
)

// ========== License API Handlers ==========

// ActivateLicenseHandler 激活 License
// POST /api/license/activate
// 注意：只能激活已存在的 License，不会自动创建
func (s *Handler) ActivateLicenseHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	err := req.ParseMultipartForm(apiFrom.MaxMemory) // maxMemory 4MB
	if err != nil {
		s.log.Errorf("Error parsing multipart form: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	licenseKey := req.PostForm.Get("license_key")
	deviceID := req.PostForm.Get("device_id")
	deviceName := req.PostForm.Get("device_name")
	deviceType := req.PostForm.Get("device_type")
	if deviceType == "" {
		deviceType = "desktop"
	}

	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	if deviceID == "" {
		s.log.Errorf("Device ID is required")
		s.jsonError(res, "Device ID is required", http.StatusBadRequest)
		return
	}

	// 查找 License (必须已存在)
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		// License 不存在，返回错误
		s.log.Errorf("Invalid license key: %s", licenseKey)
		s.jsonError(res, "Invalid license key: License not found", http.StatusNotFound)
		return
	}

	now := timestamp.CurrentTimeMillis()
	firstTimeActivation := false

	// 首次激活 - 设置日期
	if !license.Activated {
		license.Activated = true
		license.ActivatedAt = now
		firstTimeActivation = true

		// 设置 IssueDate 为当前时间
		if license.IssueDate == 0 {
			license.IssueDate = now
		}

		// 根据 License Plan 设置 ExpiryDate
		if license.ExpiryDate == 0 {
			features := contentVO.GetPlanFeatures(license.Plan)
			validityDays := features.ValidityDays
			expiryTime := time.Now().Add(time.Duration(validityDays) * 24 * time.Hour)
			license.ExpiryDate = expiryTime.UnixMilli()
		}

		if err := s.contentApp.UpdateLicense(license); err != nil {
			s.jsonError(res, "Failed to update license: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Allocate publish subdomain by default upon first activation
		if license.GetFeatures().PublishEnabled {
			_, err := s.contentApp.GetSubDomainByKey(s.db.UserDir())
			if err != nil {
				sd := &contentVO.SubDomain{
					License: license.LicenseKey,
					Sub:     s.db.UserDir(),
					Item: contentVO.Item{
						Timestamp: now,
						Updated:   now,
						Namespace: "SubDomain",
					},
				}

				if _, err := s.contentApp.CreateSubDomain(sd); err != nil {
					s.log.Errorf("Failed to create subdomain for license %s: %v", license.LicenseKey, err)
					s.jsonError(res, "Failed to create subdomain: "+err.Error(), http.StatusInternalServerError)
					return
				}

				pd := &contentVO.PublishDomain{
					License:   license.LicenseKey,
					SubDomain: sd.Sub,
					Folder:    s.db.UserDir(),
					CusDomain: "",
					Item: contentVO.Item{
						Timestamp: now,
						Updated:   now,
						Namespace: "PublishDomain",
					},
				}

				if _, err := s.contentApp.CreatePublishDomain(pd); err != nil {
					s.log.Errorf("Failed to create publish domain for license %s: %v", license.LicenseKey, err)
					s.jsonError(res, "Failed to create publish domain: "+err.Error(), http.StatusInternalServerError)
					return
				}

				if err := s.caddyClient.AddStaticSite(
					fmt.Sprintf("%s.%s", sd.Sub, s.adminApp.Domain()),
					filepath.Join(application.PreviewDir(), s.db.UserDir(), application.SubDomainFolder())); err != nil {
					s.log.Errorf("Failed to add static site to Caddy for subdomain %s: %v", sd.Sub, err)
					s.jsonError(res, "Failed to add static site to Caddy: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// 验证 License 是否过期
	if license.IsExpired() {
		s.jsonError(res, "License has expired", http.StatusForbidden)
		return
	}

	if !license.IsValid() {
		s.log.Errorf("License is not valid: %s", licenseKey)
		s.jsonError(res, "License is not valid", http.StatusForbidden)
		return
	}

	// 查找已存在的设备
	existingDevice, err := s.contentApp.GetDeviceByID(license.LicenseKey, deviceID)
	if err == nil && existingDevice != nil {
		// 设备已存在，更新访问记录
		existingDevice.LastSeenAt = now
		existingDevice.AccessCount++
		if err := s.contentApp.UpdateDevice(existingDevice); err != nil {
			s.log.Errorf("Failed to update device: %v", err)
			s.jsonError(res, "Failed to update device: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// 新设备 - 检查限制
		if !license.CanAddDevice() {
			if err := fmt.Errorf("device limit reached (%d/%d)", license.CurrentDevices, license.MaxDevices); err != nil {
				s.log.Errorf("Device limit reached for license %s: %v", license.LicenseKey, err)
				s.jsonError(res, "Device limit reached", http.StatusForbidden)
				return
			}
		}

		// 创建新设备记录
		device := &contentVO.LicenseDevice{
			License:     license.LicenseKey,
			DeviceID:    deviceID,
			DeviceName:  deviceName,
			DeviceType:  deviceType,
			FirstSeenAt: now,
			LastSeenAt:  now,
			AccessCount: 1,
			Status:      "active",
			Item: contentVO.Item{
				Timestamp: now,
				Updated:   now,
				Namespace: "LicenseDevice",
			},
		}

		if _, err := s.contentApp.CreateDevice(device); err != nil {
			s.log.Errorf("Failed to create device record: %v", err)
			s.jsonError(res, "Failed to create device record: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 更新 License 设备计数
		license.CurrentDevices++
	}

	// 获取客户端 IP
	ipAddress := s.getClientIP(req)

	// 查找已存在的 IP
	existingIP, err := s.contentApp.GetIPByAddress(license.LicenseKey, ipAddress)
	if err == nil && existingIP != nil {
		// IP 已存在，更新访问记录
		existingIP.LastSeenAt = now
		existingIP.AccessCount++
		if err := s.contentApp.UpdateLicenseIP(existingIP); err != nil {
			s.log.Errorf("Failed to update IP: %v", err)
			s.jsonError(res, "Failed to update IP: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// 新 IP - 检查限制
		if !license.CanAddIP() {
			s.log.Errorf("IP limit reached for license %s", license.LicenseKey)
			s.jsonError(res, "IP limit reached", http.StatusForbidden)
			return
		}

		// 创建新 IP 记录
		ip := &contentVO.LicenseIP{
			License:     license.LicenseKey,
			IPAddress:   ipAddress,
			FirstSeenAt: now,
			LastSeenAt:  now,
			AccessCount: 1,
			Status:      "active",
			Item: contentVO.Item{
				Timestamp: now,
				Updated:   now,
				Namespace: "LicenseIP",
			},
		}

		if _, err := s.contentApp.CreateLicenseIP(ip); err != nil {
			s.log.Errorf("Failed to create IP record: %v", err)
			s.jsonError(res, "Failed to create IP record: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 更新 License IP 计数
		license.CurrentIPs++
	}

	if err := s.contentApp.UpdateLicense(license); err != nil {
		s.log.Errorf("Failed to update license: %v", err)
		s.jsonError(res, "Failed to update license: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回响应
	response := map[string]interface{}{
		"success":     true,
		"first_time":  firstTimeActivation,
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

	// 创建 Sync 账号 (如果支持)
	var syncInfo map[string]interface{}
	if license.GetFeatures().SyncEnabled {
		syncAccount, _ := s.contentApp.GetSyncAccountByLicense(license.LicenseKey)
		if syncAccount != nil {
			// 账号已存在，返回信息（密码从 License 生成）
			syncInfo = map[string]interface{}{
				"email":       syncAccount.Email,
				"db_name":     syncAccount.DBName,
				"db_password": license.ToPassword(),
				"db_endpoint": syncAccount.DBEndpoint,
				"status":      syncAccount.Status,
			}
			response["sync"] = syncInfo

		} else {
			email := license.ToEmail()
			password := license.ToPassword()
			dbName := fmt.Sprintf("%s%s", s.adminApp.CouchDBPrefix(), license.ToUserDir())

			// 创建 CouchDB 数据库
			if err := s.couchClient.CreateDatabase(dbName); err != nil {
				s.log.Errorf("Failed to create database for sync account: %v", err)
				s.jsonError(res, "Failed to create database for sync account: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// 创建用户
			if err := s.couchClient.CreateUser(email, password); err != nil {
				s.log.Errorf("Failed to create user for sync account: %v", err)
				s.jsonError(res, "Failed to create user for sync account: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// 设置数据库权限
			if err := s.couchClient.SetDatabasePermission(dbName, email); err != nil {
				s.log.Errorf("Failed to set database permission for sync account: %v", err)
				s.jsonError(res, "Failed to set database permission for sync account: "+err.Error(), http.StatusInternalServerError)
				return
			}

			account := &contentVO.SyncAccount{
				License:    license.LicenseKey,
				Email:      email,
				DBName:     dbName,
				DBPassword: password,
				DBEndpoint: s.adminApp.CouchDBURL(),
				Status:     "active",
				CreatedAt:  now,
				Item: contentVO.Item{
					Timestamp: now,
					Updated:   now,
					Namespace: "SyncAccount",
				},
			}

			if _, err := s.contentApp.CreateSyncAccount(account); err != nil {
				s.log.Errorf("Failed to save sync account: %v", err)
				s.jsonError(res, "Failed to save sync account: "+err.Error(), http.StatusInternalServerError)
				return
			}

			syncInfo = map[string]interface{}{
				"email":       account.Email,
				"db_name":     account.DBName,
				"db_password": password,
				"db_endpoint": account.DBEndpoint,
				"status":      account.Status,
			}
			response["sync"] = syncInfo
		}
	}

	s.jsonResponse(res, response)
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
func (s *Handler) GetDevicesHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 获取设备列表
	devices, err := s.contentApp.GetDevicesByLicense(license.LicenseKey)
	if err != nil {
		s.log.Errorf("Failed to get devices for license %s: %v", license.LicenseKey, err)
		s.jsonError(res, "Failed to get devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 构建响应
	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"device_id":     device.DeviceID,
			"device_name":   device.DeviceName,
			"device_type":   device.DeviceType,
			"first_seen_at": device.FirstSeenAt,
			"last_seen_at":  device.LastSeenAt,
			"access_count":  device.AccessCount,
			"status":        device.Status,
		})
	}

	s.jsonResponse(res, map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
	})
}

// GetIPsHandler 获取 License 的 IP 列表
// GET /api/license/ips?key=xxx
func (s *Handler) GetIPsHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 获取 IP 列表
	ips, err := s.contentApp.GetIPsByLicense(license.LicenseKey)
	if err != nil {
		s.log.Errorf("Failed to get IPs for license %s: %v", license.LicenseKey, err)
		s.jsonError(res, "Failed to get IPs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 构建响应
	ipList := make([]map[string]interface{}, 0, len(ips))
	for _, ip := range ips {
		ipList = append(ipList, map[string]interface{}{
			"ip_address":    ip.IPAddress,
			"country":       ip.Country,
			"region":        ip.Region,
			"city":          ip.City,
			"first_seen_at": ip.FirstSeenAt,
			"last_seen_at":  ip.LastSeenAt,
			"access_count":  ip.AccessCount,
			"status":        ip.Status,
		})
	}

	s.jsonResponse(res, map[string]interface{}{
		"ips":   ipList,
		"count": len(ipList),
	})
}

// GetSyncInfoHandler 获取 Sync 信息
// GET /api/license/sync?key=xxx
func (s *Handler) GetSyncInfoHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 检查是否支持 Sync 功能
	if !license.GetFeatures().SyncEnabled {
		s.jsonError(res, "Sync feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	// 获取 Sync 账号
	account, err := s.contentApp.GetSyncAccountByLicense(license.LicenseKey)
	if err != nil {
		s.log.Errorf("Sync account not found for license %s: %v", license.LicenseKey, err)
		s.jsonError(res, "Sync account not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"email":       account.Email,
		"db_name":     account.DBName,
		"db_endpoint": account.DBEndpoint,
		"status":      account.Status,
		"created_at":  account.CreatedAt,
	}

	s.jsonResponse(res, response)
}

// GetPublishInfoHandler 获取 Publish 信息
// GET /api/license/publish?key=xxx
func (s *Handler) GetPublishInfoHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 检查是否支持 Publish 功能
	if !license.GetFeatures().PublishEnabled {
		s.jsonError(res, "Publish feature not enabled for this license plan", http.StatusForbidden)
		return
	}

	response := map[string]interface{}{
		"enabled": true,
		"plan":    license.Plan,
		"features": map[string]interface{}{
			"max_sites":      license.GetFeatures().MaxSites,
			"max_storage_mb": license.GetFeatures().MaxStorageMB,
			"custom_domain":  license.GetFeatures().CustomDomain,
		},
	}

	s.jsonResponse(res, response)
}

// GetDisksHandler 获取硬盘用量信息
// GET /api/license/disks?key=xxx
func (s *Handler) GetDisksHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(res, s.getDisksUsage(license.LicenseKey))
}

// getDisksUsage 内部方法：获取硬盘用量信息
func (s *Handler) getDisksUsage(licenseKey string) map[string]interface{} {
	var couchDBDiskUsageMB float64 = 0
	var publishDiskUsageMB float64 = 0

	// 获取 CouchDB 磁盘用量
	syncAccount, err := s.contentApp.GetSyncAccountByLicense(licenseKey)
	if err == nil && syncAccount != nil {
		// 获取 CouchDB 数据库磁盘用量
		diskUsageBytes, err := s.couchClient.GetDiskUsage(syncAccount.DBName)
		if err != nil {
			s.log.Errorf("Failed to get CouchDB disk usage for %s: %v", syncAccount.DBName, err)
			// 不返回错误，继续获取其他信息
		} else {
			couchDBDiskUsageMB = float64(diskUsageBytes) / (1024 * 1024)
		}
	}

	// 获取 Publish 目录磁盘用量
	publishDir := filepath.Join(application.PreviewDir(), s.db.UserDir())
	publishDiskUsageBytes, err := dirSizeByDU(publishDir)
	if err != nil {
		s.log.Errorf("Failed to get publish disk usage for %s: %v", publishDir, err)
		// 不返回错误，继续
	} else {
		publishDiskUsageMB = float64(publishDiskUsageBytes) / (1024 * 1024)
	}

	// 计算总用量
	totalDiskUsageMB := couchDBDiskUsageMB + publishDiskUsageMB

	return map[string]interface{}{
		"couchdb_disk_usage": fmt.Sprintf("%.2f", couchDBDiskUsageMB),
		"publish_disk_usage": fmt.Sprintf("%.2f", publishDiskUsageMB),
		"total_disk_usage":   fmt.Sprintf("%.2f", totalDiskUsageMB),
		"unit":               "MB",
	}
}

// GetUsageHandler 获取 License 使用量汇总
// GET /api/license/usage?key=xxx
// 汇总 devices, ips, disks 三个接口的信息
func (s *Handler) GetUsageHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	licenseKey := req.URL.Query().Get("key")
	if licenseKey == "" {
		s.log.Errorf("License key is required")
		s.jsonError(res, "License key is required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 获取设备信息
	devicesInfo := s.getDevicesInfo(license.LicenseKey)

	// 获取 IP 信息
	ipsInfo := s.getIPsInfo(license.LicenseKey)

	// 获取硬盘用量信息
	disksInfo := s.getDisksUsage(license.LicenseKey)

	// 汇总响应
	response := map[string]interface{}{
		"license_key": license.LicenseKey,
		"plan":        license.Plan,
		"features":    license.GetFeatures(),
		"devices":     devicesInfo,
		"ips":         ipsInfo,
		"disks":       disksInfo,
	}

	s.jsonResponse(res, response)
}

// getDevicesInfo 内部方法：获取设备信息
func (s *Handler) getDevicesInfo(licenseKey string) map[string]interface{} {
	devices, err := s.contentApp.GetDevicesByLicense(licenseKey)
	if err != nil {
		s.log.Errorf("Failed to get devices for license %s: %v", licenseKey, err)
		return map[string]interface{}{
			"devices": []map[string]interface{}{},
			"count":   0,
			"error":   err.Error(),
		}
	}

	deviceList := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, map[string]interface{}{
			"device_id":     device.DeviceID,
			"device_name":   device.DeviceName,
			"device_type":   device.DeviceType,
			"first_seen_at": device.FirstSeenAt,
			"last_seen_at":  device.LastSeenAt,
			"access_count":  device.AccessCount,
			"status":        device.Status,
		})
	}

	return map[string]interface{}{
		"devices": deviceList,
		"count":   len(deviceList),
	}
}

// getIPsInfo 内部方法：获取 IP 信息
func (s *Handler) getIPsInfo(licenseKey string) map[string]interface{} {
	ips, err := s.contentApp.GetIPsByLicense(licenseKey)
	if err != nil {
		s.log.Errorf("Failed to get IPs for license %s: %v", licenseKey, err)
		return map[string]interface{}{
			"ips":   []map[string]interface{}{},
			"count": 0,
			"error": err.Error(),
		}
	}

	ipList := make([]map[string]interface{}, 0, len(ips))
	for _, ip := range ips {
		ipList = append(ipList, map[string]interface{}{
			"ip_address":    ip.IPAddress,
			"country":       ip.Country,
			"region":        ip.Region,
			"city":          ip.City,
			"first_seen_at": ip.FirstSeenAt,
			"last_seen_at":  ip.LastSeenAt,
			"access_count":  ip.AccessCount,
			"status":        ip.Status,
		})
	}

	return map[string]interface{}{
		"ips":   ipList,
		"count": len(ipList),
	}
}

// dirSizeByDU 使用 du 命令获取目录大小（字节）
func dirSizeByDU(path string) (int64, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("du", "-sb", path)
	case "darwin":
		cmd = exec.Command("du", "-sk", path)
	default:
		return 0, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, fmt.Errorf("unexpected du output: %s", out)
	}

	size, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, err
	}

	if runtime.GOOS == "darwin" {
		size *= 1024
	}

	return size, nil
}

// BlockDeviceHandler 封禁设备
// POST /api/license/device/block
func (s *Handler) BlockDeviceHandler(res http.ResponseWriter, req *http.Request) {
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
	deviceID := req.PostForm.Get("device_id")

	if licenseKey == "" || deviceID == "" {
		s.log.Errorf("License key and device ID are required")
		s.jsonError(res, "License key and device ID are required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 获取设备记录
	device, err := s.contentApp.GetDeviceByID(license.LicenseKey, deviceID)
	if err != nil {
		s.log.Errorf("Device not found: %s", deviceID)
		s.jsonError(res, "Device not found", http.StatusNotFound)
		return
	}

	// 更新设备状态为 blocked
	device.Status = "blocked"
	if err := s.contentApp.UpdateDevice(device); err != nil {
		s.log.Errorf("Failed to block device: %v", err)
		s.jsonError(res, "Failed to block device: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
		"success": true,
		"message": "Device blocked successfully",
	})
}

// BlockIPHandler 封禁 IP
// POST /api/license/ip/block
func (s *Handler) BlockIPHandler(res http.ResponseWriter, req *http.Request) {
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
	ipAddress := req.PostForm.Get("ip_address")

	if licenseKey == "" || ipAddress == "" {
		s.log.Errorf("License key and IP address are required")
		s.jsonError(res, "License key and IP address are required", http.StatusBadRequest)
		return
	}

	// 验证 License 是否存在
	license, err := s.contentApp.GetLicenseByKey(licenseKey)
	if err != nil {
		s.log.Errorf("License not found: %s", licenseKey)
		s.jsonError(res, "License not found", http.StatusNotFound)
		return
	}

	// 获取 IP 记录
	ip, err := s.contentApp.GetIPByAddress(license.LicenseKey, ipAddress)
	if err != nil {
		s.log.Errorf("IP not found: %s", ipAddress)
		s.jsonError(res, "IP not found", http.StatusNotFound)
		return
	}

	// 更新 IP 状态为 blocked
	ip.Status = "blocked"
	if err := s.contentApp.UpdateLicenseIP(ip); err != nil {
		s.log.Errorf("Failed to block IP: %v", err)
		s.jsonError(res, "Failed to block IP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(res, map[string]interface{}{
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
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		h.log.Errorf("Error marshalling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err := h.res.FmtJSON(jsonBytes)
	if err != nil {
		h.log.Errorf("Error formatting JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.res.Json(w, j)
}

func (h *Handler) jsonError(w http.ResponseWriter, message string, status int) {
	errorData := map[string]interface{}{
		"success": false,
		"error":   message,
	}

	jsonBytes, err := json.Marshal(errorData)
	if err != nil {
		h.log.Errorf("Error marshalling error JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err := h.res.FmtJSON(jsonBytes)
	if err != nil {
		h.log.Errorf("Error formatting error JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	h.res.Json(w, j)
}
