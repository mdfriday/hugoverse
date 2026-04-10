package licensekit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MasterLicenseInfo Master License 信息（从 API 返回）
type MasterLicenseInfo struct {
	Valid          bool      `json:"valid"`            // 是否有效
	LicenseKey     string    `json:"license_key"`      // License Key
	Type           string    `json:"type"`             // 类型：free, starter, pro, unlimited
	MaxSubLicenses int       `json:"max_sub_licenses"` // 最大子 License 数量
	UsedLicenses   int       `json:"used_licenses"`    // 已使用数量
	ExpiryDate     time.Time `json:"expiry_date"`      // 过期时间
	Holder         string    `json:"holder"`           // 持有者
	Features       []string  `json:"features"`         // 功能列表
	Message        string    `json:"message"`          // 消息（错误信息）
}

// VerifyMasterLicenseOnline 在线验证 Master License
func VerifyMasterLicenseOnline(masterLicense string) (*MasterLicenseInfo, error) {
	// 如果没有提供 Master License，返回免费版配置
	if masterLicense == "" {
		return GetFreeMasterLicenseInfo(), nil
	}

	// 调用官方 API 验证
	apiURL := "https://api.mdfriday.com/v1/master-license/verify"

	requestBody := map[string]string{
		"license_key": masterLicense,
		"version":     "1.0",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Hugoverse/1.0")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// 网络错误，降级到免费版
		return GetFreeMasterLicenseInfo(), fmt.Errorf("failed to verify online (using free mode): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetFreeMasterLicenseInfo(), fmt.Errorf("failed to read response (using free mode): %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return GetFreeMasterLicenseInfo(), fmt.Errorf("verification failed (status %d, using free mode): %s", resp.StatusCode, string(body))
	}

	var info MasterLicenseInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return GetFreeMasterLicenseInfo(), fmt.Errorf("failed to parse response (using free mode): %w", err)
	}

	if !info.Valid {
		// License 无效，降级到免费版
		return GetFreeMasterLicenseInfo(), fmt.Errorf("master license is invalid: %s", info.Message)
	}

	return &info, nil
}

// GetFreeMasterLicenseInfo 获取免费版配置
func GetFreeMasterLicenseInfo() *MasterLicenseInfo {
	return &MasterLicenseInfo{
		Valid:          true,
		LicenseKey:     "FREE",
		Type:           "free",
		MaxSubLicenses: 1,
		UsedLicenses:   0,
		ExpiryDate:     time.Now().AddDate(100, 0, 0), // 100 年
		Holder:         "free-user",
		Features:       []string{"enterprise_license"},
		Message:        "Free version - 1 enterprise license included",
	}
}

// CanGenerateMore 检查是否还能生成更多 License
func (ml *MasterLicenseInfo) CanGenerateMore(requestCount int) bool {
	return (ml.UsedLicenses + requestCount) <= ml.MaxSubLicenses
}

// GetRemainingQuota 获取剩余配额
func (ml *MasterLicenseInfo) GetRemainingQuota() int {
	remaining := ml.MaxSubLicenses - ml.UsedLicenses
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExpired 检查是否过期
func (ml *MasterLicenseInfo) IsExpired() bool {
	return time.Now().After(ml.ExpiryDate)
}

// ReportUsage 上报使用情况（生成 License 后调用）
func ReportUsage(masterLicense string, generatedCount int) error {
	if masterLicense == "" || masterLicense == "FREE" {
		// 免费版不需要上报
		return nil
	}

	apiURL := "https://api.mdfriday.com/v1/master-license/report-usage"

	requestBody := map[string]interface{}{
		"license_key": masterLicense,
		"count":       generatedCount,
		"timestamp":   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Hugoverse/1.0")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// 上报失败不阻塞流程
		return fmt.Errorf("failed to report usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
