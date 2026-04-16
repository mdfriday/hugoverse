package handler

import (
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/pkg/version"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mdfriday/hugoverse/internal/application"
	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
)

// ========== Instance API Handlers ==========

// CreateInstanceHandler 创建实例
// POST /api/instance/create
func (s *Handler) CreateInstanceHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 解析 form 数据（与 License API 一致）
	err := req.ParseForm()
	if err != nil {
		s.log.Errorf("Error parsing form: %v", err)
		s.jsonError(res, "Invalid request", http.StatusBadRequest)
		return
	}

	instanceID := req.PostForm.Get("instance_id")
	domain := req.PostForm.Get("domain")
	version := req.PostForm.Get("version")
	ipAddress := req.PostForm.Get("ip_address")
	userAgent := req.PostForm.Get("user_agent")
	status := req.PostForm.Get("status")

	// 解析数字字段
	totalLicenses := 0
	if val := req.PostForm.Get("total_licenses"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			totalLicenses = parsed
		}
	}

	totalTrials := 0
	if val := req.PostForm.Get("total_trials"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			totalTrials = parsed
		}
	}

	allowOfflineSeconds := int64(0)
	if val := req.PostForm.Get("allow_offline_seconds"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			allowOfflineSeconds = parsed
		}
	}

	// 验证必填字段
	if instanceID == "" {
		s.jsonError(res, "instance_id is required", http.StatusBadRequest)
		return
	}

	// 检查实例是否已存在
	existingInstance, err := s.contentApp.GetInstanceByID(instanceID)
	if err == nil && existingInstance != nil {
		// 实例已存在，返回现有实例
		s.log.Printf("Instance already exists: %s", instanceID)
		response := map[string]interface{}{
			"success":  true,
			"message":  "Instance already exists",
			"instance": existingInstance,
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(response)
		return
	}

	// 创建新实例
	now := time.Now().Unix()
	instance := &contentVO.Instance{
		InstanceID:          instanceID,
		Domain:              domain,
		TotalLicenses:       totalLicenses,
		TotalTrials:         totalTrials,
		Version:             version,
		IPAddress:           ipAddress,
		UserAgent:           userAgent,
		Status:              contentVO.InstanceStatus(status),
		LastSeenAt:          now,
		CreatedAt:           now,
		AllowOfflineSeconds: allowOfflineSeconds,
		Item: contentVO.Item{
			Timestamp: now,
			Updated:   now,
			Namespace: "Instance",
		},
	}

	// 设置默认值
	if instance.Status == "" {
		instance.Status = contentVO.InstanceActive
	}
	if instance.AllowOfflineSeconds == 0 {
		instance.AllowOfflineSeconds = 180 * 24 * 60 * 60 // 半年
	}

	// 保存到数据库
	if err := s.contentApp.CreateInstance(instance); err != nil {
		s.log.Errorf("Failed to create instance: %v", err)
		s.jsonError(res, fmt.Sprintf("Failed to create instance: %v", err), http.StatusInternalServerError)
		return
	}

	s.log.Printf("✅ Instance created: %s", instance.InstanceID)

	response := map[string]interface{}{
		"success":  true,
		"message":  "Instance created successfully",
		"instance": instance,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(response)
}

// GetInstanceHandler 查询实例
// GET /api/instance?instance_id=xxx
func (s *Handler) GetInstanceHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	instanceID := req.URL.Query().Get("instance_id")
	if instanceID == "" {
		s.jsonError(res, "instance_id is required", http.StatusBadRequest)
		return
	}

	instance, err := s.contentApp.GetInstanceByID(instanceID)
	if err != nil {
		s.log.Errorf("Failed to get instance: %v", err)
		s.jsonError(res, "Instance not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"instance": instance,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(response)
}

// UpdateInstanceHandler 更新实例
// POST /api/instance/update
func (s *Handler) UpdateInstanceHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 解析 form 数据（与 License API 一致）
	err := req.ParseForm()
	if err != nil {
		s.log.Errorf("Error parsing form: %v", err)
		s.jsonError(res, "Invalid request", http.StatusBadRequest)
		return
	}

	instanceID := req.PostForm.Get("instance_id")
	if instanceID == "" {
		s.jsonError(res, "instance_id is required", http.StatusBadRequest)
		return
	}

	// 获取现有实例
	instance, err := s.contentApp.GetInstanceByID(instanceID)
	if err != nil {
		s.log.Errorf("Failed to get instance: %v", err)
		s.jsonError(res, "Instance not found", http.StatusNotFound)
		return
	}

	// 更新字段（只更新提供的字段）
	if val := req.PostForm.Get("total_licenses"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			instance.TotalLicenses = parsed
		}
	}
	if val := req.PostForm.Get("total_trials"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			instance.TotalTrials = parsed
		}
	}
	if val := req.PostForm.Get("version"); val != "" {
		instance.Version = val
	}
	if val := req.PostForm.Get("ip_address"); val != "" {
		instance.IPAddress = val
	}
	if val := req.PostForm.Get("user_agent"); val != "" {
		instance.UserAgent = val
	}
	if val := req.PostForm.Get("status"); val != "" {
		instance.Status = contentVO.InstanceStatus(val)
	}
	if val := req.PostForm.Get("allow_offline_seconds"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			instance.AllowOfflineSeconds = parsed
		}
	}

	// 更新心跳时间和 Item 时间戳
	now := time.Now().Unix()
	instance.LastSeenAt = now
	instance.Item.Updated = now

	// 保存到数据库
	if err := s.contentApp.UpdateInstance(instance); err != nil {
		s.log.Errorf("Failed to update instance: %v", err)
		s.jsonError(res, fmt.Sprintf("Failed to update instance: %v", err), http.StatusInternalServerError)
		return
	}

	s.log.Printf("✅ Instance updated: %s", instance.InstanceID)

	response := map[string]interface{}{
		"success":  true,
		"message":  "Instance updated successfully",
		"instance": instance,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(response)
}

// syncInstanceFromRemote 从远端服务器同步实例状态
// MDFriday 使用相同的代码库，提供相同的接口
func (s *Handler) syncInstanceFromRemote(instanceID string) (*contentVO.Instance, error) {
	apiBase := "https://app.mdfriday.com"

	// 调用远端查询 API
	resp, err := http.Get(fmt.Sprintf("%s/api/instance?instance_id=%s", apiBase, instanceID))
	if err != nil {
		return nil, fmt.Errorf("failed to call remote API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Success  bool                `json:"success"`
		Instance *contentVO.Instance `json:"instance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success || response.Instance == nil {
		return nil, fmt.Errorf("remote API call failed or instance not found")
	}

	return response.Instance, nil
}

func (s *Handler) CheckInstanceStatus(next http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// 获取 InstanceManager
		instanceMgr := application.NewInstanceManager(s.log, version.CurrentVersion.String())
		
		// 检测本地开发环境，直接放行
		if instanceMgr.IsLocalDevelopment() {
			s.log.Println("Local development environment detected, skipping instance check")
			next(res, req)
			return
		}
		
		localData, err := instanceMgr.GetLocalInstance()

		if err != nil || localData == nil {
			s.log.Warnf("Failed to get local instance data: %v", err)
			// 本地数据不存在，继续执行（首次运行的情况）
			next(res, req)
			return
		}

		// 策略1: 检查本地缓存是否有效（24小时内）
		if instanceMgr.IsCacheValid() {
			s.log.Debugf("Using cached instance data (valid for 24h)")

			// 使用本地缓存数据判断
			if localData.Status == string(contentVO.InstanceBlocked) {
				s.jsonError(res, "Instance is blocked", http.StatusForbidden)
				return
			}

			// 缓存有效，继续执行
			next(res, req)
			return
		}

		// 策略2: 缓存过期，尝试从远端同步
		s.log.Debugf("Cache expired, syncing from remote server...")
		remoteInstance, err := s.syncInstanceFromRemote(localData.InstanceID)

		if err != nil {
			// 远端同步失败，检查离线时间
			s.log.Warnf("Failed to sync from remote: %v", err)

			// 检查是否超过允许的离线时间（半年）
			if err := instanceMgr.CheckOfflineStatus(); err != nil {
				s.log.Errorf("Instance offline check failed: %v", err)
				s.jsonError(res,
					"Unable to connect to the app.mdfriday.com server; offline time has exceeded the limit. Please ensure the server can access app.mdfriday.com, and then try again.",
					http.StatusForbidden)
				return
			}

			// 离线时间在允许范围内，使用本地状态继续
			if localData.Status == string(contentVO.InstanceBlocked) {
				s.jsonError(res, "Instance is blocked", http.StatusForbidden)
				return
			}

			next(res, req)
			return
		}

		// 策略3: 远端同步成功，更新本地缓存
		s.log.Debugf("✅ Remote sync successful, updating local cache")
		if err := instanceMgr.UpdateFromRemote(remoteInstance); err != nil {
			s.log.Warnf("Failed to update local cache: %v", err)
		}

		// 检查远端返回的状态
		if remoteInstance.Status == contentVO.InstanceBlocked {
			s.jsonError(res, "Instance is blocked by server", http.StatusForbidden)
			return
		}

		// 异步更新远端统计数据（不阻塞主流程）
		go s.updateRemoteInstanceStats(localData.InstanceID)

		// 继续执行下一个处理器
		next(res, req)
	}
}

// updateInstanceStats 更新实例统计信息
// 统计当前系统中的 license 和 trial 数量，并更新到本地 JSON 文件
func (s *Handler) updateInstanceStats() {
	// 获取 InstanceManager
	instanceMgr := application.NewInstanceManager(s.log, "26.4.1") // TODO: 从配置获取版本
	localData, err := instanceMgr.GetLocalInstance()

	if err != nil || localData == nil {
		s.log.Warnf("Failed to get local instance for stats update: %v", err)
		return
	}

	// 统计激活的 licenses
	allLicenses := s.contentApp.AllContents("License")
	activatedCount := 0
	for _, licenseData := range allLicenses {
		var license contentVO.License
		if err := json.Unmarshal(licenseData, &license); err != nil {
			continue
		}
		if license.Activated && !license.IsExpired() {
			activatedCount++
		}
	}

	// 统计 trials
	allTrials := s.contentApp.AllContents("LicenseTrial")
	trialCount := len(allTrials)

	// 检查是否需要更新
	needUpdate := false
	if localData.TotalLicenses != activatedCount {
		localData.TotalLicenses = activatedCount
		needUpdate = true
	}
	if localData.TotalTrials != trialCount {
		localData.TotalTrials = trialCount
		needUpdate = true
	}

	if needUpdate {
		now := time.Now().Unix()
		localData.LastSeenAt = now

		// 直接更新本地 JSON 文件
		if err := instanceMgr.SaveLocalDataDirect(localData); err != nil {
			s.log.Warnf("Failed to update local instance file: %v", err)
		} else {
			s.log.Printf("✅ Instance stats updated: TotalLicenses=%d, TotalTrials=%d",
				localData.TotalLicenses, localData.TotalTrials)
		}
	}
}

// updateRemoteInstanceStats 异步更新远端实例统计数据
// 统计当前运行实例的真实 license 和 trial 数量，并同步到远端数据库
func (s *Handler) updateRemoteInstanceStats(instanceID string) {
	// 检测本地开发环境，跳过远端更新
	instanceMgr := application.NewInstanceManager(s.log, version.CurrentVersion.String())
	if instanceMgr.IsLocalDevelopment() {
		s.log.Println("Local development environment detected, skipping remote stats update")
		return
	}
	
	// 统计激活的 licenses
	allLicenses := s.contentApp.AllContents("License")
	licenseCount := len(allLicenses)

	// 统计 trials
	allTrials := s.contentApp.AllContents("LicenseTrial")
	trialCount := len(allTrials)

	// 构建 form 数据（与 License API 一致）
	apiBase := "https://app.mdfriday.com"
	formData := url.Values{}
	formData.Set("instance_id", instanceID)
	formData.Set("total_licenses", strconv.Itoa(licenseCount))
	formData.Set("total_trials", strconv.Itoa(trialCount))

	// 调用远端更新 API
	resp, err := http.Post(
		apiBase+"/api/instance/update",
		"application/x-www-form-urlencoded",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		s.log.Warnf("Failed to update remote instance stats: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.log.Warnf("Remote API returned status %d: %s", resp.StatusCode, string(body))
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		s.log.Warnf("Failed to decode update response: %v", err)
		return
	}

	if success, ok := response["success"].(bool); ok && success {
		s.log.Debugf("✅ Remote instance stats updated: TotalLicenses=%d, TotalTrials=%d",
			licenseCount, trialCount)
	} else {
		s.log.Warnf("Remote instance stats update failed: %v", response)
	}
}
