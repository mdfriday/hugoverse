package application

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	contentVO "github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

const (
	instanceFileName  = "instance.json"
	halfYearSeconds   = 180 * 24 * 60 * 60             // 半年的秒数
	cacheValidSeconds = 24 * 60 * 60                   // 缓存有效期：24小时
	instanceSalt      = "hugoverse-instance-salt-2024" // 用于生成 instance_id 的盐
	remoteInstanceAPI = "https://app.mdfriday.com"     // 远端 Instance API 地址
)

// InstanceManager 实例管理器
type InstanceManager struct {
	log       loggers.Logger
	version   string
	localPath string // 本地 JSON 文件路径
}

// NewInstanceManager 创建实例管理器
func NewInstanceManager(log loggers.Logger, version string) *InstanceManager {
	localPath := filepath.Join(MDFridayDir(), instanceFileName)

	return &InstanceManager{
		log:       log,
		version:   version,
		localPath: localPath,
	}
}

// LocalInstanceData 本地实例数据（持久化到 JSON）
type LocalInstanceData struct {
	InstanceID          string `json:"instance_id"`
	Domain              string `json:"domain"`                // 实例域名
	LastSeenAt          int64  `json:"last_seen_at"`          // 最后心跳时间（用于判断离线）
	LastSyncAt          int64  `json:"last_sync_at"`          // 最后同步时间（用于判断缓存是否过期）
	CreatedAt           int64  `json:"created_at"`            // 首次安装时间
	AllowOfflineSeconds int64  `json:"allow_offline_seconds"` // 允许离线时间（默认半年）
	Status              string `json:"status"`                // 状态（active/blocked/suspended）
	TotalLicenses       int    `json:"total_licenses"`        // 总 license 数量
	TotalTrials         int    `json:"total_trials"`          // 总 trial 数量
}

// GetOrCreateInstance 获取或创建实例
// 首先检查本地 JSON 文件，如果不存在则创建新实例
func (m *InstanceManager) GetOrCreateInstance() (*contentVO.Instance, error) {
	// 1. 尝试从本地文件加载
	localData, err := m.loadLocalData()
	if err == nil && localData != nil {
		m.log.Printf("Loaded instance from local file: %s", localData.InstanceID)

		// 检查是否需要更新（心跳检查）
		now := time.Now().Unix()
		if localData.LastSeenAt > 0 && localData.AllowOfflineSeconds > 0 {
			offlineDuration := now - localData.LastSeenAt
			if offlineDuration > localData.AllowOfflineSeconds {
				return nil, fmt.Errorf("instance offline too long: %d seconds (allowed: %d)",
					offlineDuration, localData.AllowOfflineSeconds)
			}
		}

		return m.buildInstanceFromLocal(localData), nil
	}

	// 2. 本地文件不存在，生成新的 instance_id
	instanceID, err := m.generateInstanceID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate instance_id: %w", err)
	}

	// 3. 创建新实例
	now := time.Now().Unix()
	instance := &contentVO.Instance{
		InstanceID:          instanceID,
		Domain:              m.getDomain(),
		TotalLicenses:       0,
		TotalTrials:         0,
		Version:             m.version,
		IPAddress:           m.getServerIP(),
		UserAgent:           m.getUserAgent(),
		Status:              contentVO.InstanceActive,
		LastSeenAt:          now,
		CreatedAt:           now,
		AllowOfflineSeconds: halfYearSeconds,
		Item: contentVO.Item{
			Timestamp: now,
			Updated:   now,
			Namespace: "Instance",
		},
	}

	// 4. 保存到本地文件
	if err := m.saveLocalData(instance); err != nil {
		m.log.Warnf("Failed to save instance to local file: %v", err)
	}

	return instance, nil
}

// UpdateLocalInstance 更新本地实例数据
func (m *InstanceManager) UpdateLocalInstance(instance *contentVO.Instance) error {
	return m.saveLocalData(instance)
}

// GetLocalInstance 获取本地实例数据
func (m *InstanceManager) GetLocalInstance() (*LocalInstanceData, error) {
	return m.loadLocalData()
}

// IsLocalDevelopment 检测是否为本地开发环境
func (m *InstanceManager) IsLocalDevelopment() bool {
	domain := m.getDomain()
	if domain == "" {
		return false
	}

	// 检测本地开发环境的域名
	localPatterns := []string{
		"localhost",
		"127.0.0.1",
		"::1",
	}

	for _, pattern := range localPatterns {
		if domain == pattern || strings.HasSuffix(domain, ".localhost") {
			return true
		}
	}

	return false
}

// getDomain 获取实例域名
func (m *InstanceManager) getDomain() string {
	return getEnvOrDefault("DOMAIN", "localhost")
}

// CheckOfflineStatus 检查实例是否离线超时
func (m *InstanceManager) CheckOfflineStatus() error {
	localData, err := m.loadLocalData()
	if err != nil {
		return fmt.Errorf("failed to load local instance data: %w", err)
	}

	if localData == nil {
		return fmt.Errorf("no local instance data found")
	}

	now := time.Now().Unix()
	if localData.LastSeenAt > 0 && localData.AllowOfflineSeconds > 0 {
		offlineDuration := now - localData.LastSeenAt
		if offlineDuration > localData.AllowOfflineSeconds {
			return fmt.Errorf("instance offline too long: %d seconds (allowed: %d)",
				offlineDuration, localData.AllowOfflineSeconds)
		}
	}

	return nil
}

// IsCacheValid 检查本地缓存是否有效（24小时内）
func (m *InstanceManager) IsCacheValid() bool {
	localData, err := m.loadLocalData()
	if err != nil || localData == nil {
		return false
	}

	// 如果从未同步过，缓存无效
	if localData.LastSyncAt == 0 {
		return false
	}

	now := time.Now().Unix()
	elapsed := now - localData.LastSyncAt
	return elapsed < cacheValidSeconds
}

// UpdateFromRemote 从远端更新实例数据
func (m *InstanceManager) UpdateFromRemote(remoteInstance *contentVO.Instance) error {
	localData, err := m.loadLocalData()
	if err != nil || localData == nil {
		return fmt.Errorf("failed to load local data: %w", err)
	}

	// 更新本地数据
	now := time.Now().Unix()
	localData.Status = string(remoteInstance.Status)
	localData.TotalLicenses = remoteInstance.TotalLicenses
	localData.TotalTrials = remoteInstance.TotalTrials
	localData.AllowOfflineSeconds = remoteInstance.AllowOfflineSeconds
	localData.LastSeenAt = now
	localData.LastSyncAt = now

	// 保存到文件
	return m.SaveLocalDataDirect(localData)
}

// generateInstanceID 生成唯一且稳定的 instance_id
// 使用 SHA256(machine_id + install_time + salt) 确保在容器中也能保持稳定
func (m *InstanceManager) generateInstanceID() (string, error) {
	machineID, err := m.getMachineID()
	if err != nil {
		return "", fmt.Errorf("failed to get machine_id: %w", err)
	}

	// 获取或生成安装时间戳
	installTime := m.getInstallTime()

	// 使用 SHA256 生成唯一标识: SHA256(machine_id + install_time + salt)
	data := fmt.Sprintf("%s|%d|%s", machineID, installTime, instanceSalt)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash), nil
}

// getInstallTime 获取安装时间戳
// 从本地文件读取，如果不存在则生成新的时间戳
func (m *InstanceManager) getInstallTime() int64 {
	// 尝试从本地文件读取安装时间
	localData, err := m.loadLocalData()
	if err == nil && localData != nil && localData.CreatedAt > 0 {
		return localData.CreatedAt
	}

	// 生成新的安装时间戳（Unix 时间戳，秒级）
	return time.Now().Unix()
}

// getMachineID 获取机器唯一标识
// 综合多个系统特征生成稳定的 machine ID
func (m *InstanceManager) getMachineID() (string, error) {
	parts := []string{}

	// 2. 读取系统 machine-id（Linux 标准）
	if id := m.readSystemFile("/etc/machine-id"); id != "" {
		parts = append(parts, id)
	}

	if id := m.readSystemFile("/var/lib/dbus/machine-id"); id != "" {
		parts = append(parts, id)
	}

	// 3. 获取真实网卡的 MAC 地址（过滤虚拟网卡）
	if mac := m.getRealMacAddress(); mac != "" {
		parts = append(parts, mac)
	}

	// 4. 使用主机名作为补充特征
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		parts = append(parts, hostname)
	}

	// 5. 使用数据目录的绝对路径（在 Docker 中，挂载的卷路径是稳定的）
	dataDir := getEnvOrDefault("HUGOVERSE_DATA_DIR", "/data")
	if absPath, err := filepath.Abs(dataDir); err == nil && absPath != "" {
		parts = append(parts, absPath)
	}

	// 6. 如果所有特征都获取失败，返回错误
	if len(parts) == 0 {
		return "", fmt.Errorf("unable to determine any machine characteristics")
	}

	// 拼接所有特征
	return strings.Join(parts, "|"), nil
}

// readSystemFile 读取系统文件内容
func (m *InstanceManager) readSystemFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// getRealMacAddress 获取真实网卡的 MAC 地址（过滤虚拟网卡）
func (m *InstanceManager) getRealMacAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		// 跳过 loopback 接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 跳过虚拟网卡（docker, veth, bridge 等）
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "docker") ||
			strings.Contains(name, "veth") ||
			strings.Contains(name, "br-") ||
			strings.Contains(name, "virbr") {
			continue
		}

		// 获取 MAC 地址
		mac := iface.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" {
			return mac
		}
	}

	return ""
}

// getServerIP 获取服务器真实 IP 地址
func (m *InstanceManager) getServerIP() string {
	// 1. 优先使用环境变量指定的 IP
	if serverIP := os.Getenv("SERVER_IP"); serverIP != "" {
		return serverIP
	}

	// 2. 尝试获取对外网卡的 IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}

	// 3. 获取第一个非回环地址
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String()
				}
			}
		}
	}

	return "unknown"
}

// getUserAgent 获取用户代理信息
func (m *InstanceManager) getUserAgent() string {
	return fmt.Sprintf("Hugoverse/%s (%s; %s)", m.version, runtime.GOOS, runtime.GOARCH)
}

// loadLocalData 从本地 JSON 文件加载实例数据
func (m *InstanceManager) loadLocalData() (*LocalInstanceData, error) {
	data, err := os.ReadFile(m.localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在，返回 nil
		}
		return nil, fmt.Errorf("failed to read local instance file: %w", err)
	}

	var localData LocalInstanceData
	if err := json.Unmarshal(data, &localData); err != nil {
		return nil, fmt.Errorf("failed to parse local instance file: %w", err)
	}

	return &localData, nil
}

// saveLocalData 保存实例数据到本地 JSON 文件
func (m *InstanceManager) saveLocalData(instance *contentVO.Instance) error {
	now := time.Now().Unix()
	localData := &LocalInstanceData{
		InstanceID:          instance.InstanceID,
		Domain:              instance.Domain,
		LastSeenAt:          instance.LastSeenAt,
		LastSyncAt:          now, // 更新同步时间
		CreatedAt:           instance.CreatedAt,
		AllowOfflineSeconds: instance.AllowOfflineSeconds,
		Status:              string(instance.Status),
		TotalLicenses:       instance.TotalLicenses,
		TotalTrials:         instance.TotalTrials,
	}

	return m.SaveLocalDataDirect(localData)
}

// SaveLocalDataDirect 直接保存 LocalInstanceData 到文件（公开方法）
func (m *InstanceManager) SaveLocalDataDirect(localData *LocalInstanceData) error {
	data, err := json.MarshalIndent(localData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(m.localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(m.localPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write instance file: %w", err)
	}

	return nil
}

// buildInstanceFromLocal 从本地数据构建 Instance 对象
func (m *InstanceManager) buildInstanceFromLocal(localData *LocalInstanceData) *contentVO.Instance {
	now := time.Now().Unix()
	return &contentVO.Instance{
		InstanceID:          localData.InstanceID,
		Domain:              localData.Domain,
		TotalLicenses:       localData.TotalLicenses,
		TotalTrials:         localData.TotalTrials,
		Version:             m.version,
		IPAddress:           m.getServerIP(),
		UserAgent:           m.getUserAgent(),
		Status:              contentVO.InstanceStatus(localData.Status),
		LastSeenAt:          localData.LastSeenAt,
		CreatedAt:           localData.CreatedAt,
		AllowOfflineSeconds: localData.AllowOfflineSeconds,
		Item: contentVO.Item{
			Timestamp: localData.CreatedAt,
			Updated:   now,
			Namespace: "Instance",
		},
	}
}
