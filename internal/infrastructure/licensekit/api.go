package licensekit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// LoginAndGetToken 登录并获取 token
func LoginAndGetToken(apiBase, email, password string) (string, error) {
	loginURL := fmt.Sprintf("%s/api/login", apiBase)

	data := fmt.Sprintf("email=%s&password=%s", email, password)
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
		Error   string   `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) > 0 {
		return result.Data[0], nil
	}

	return "", fmt.Errorf("no token returned")
}

// CreateLicenseViaAPI 通过 API 创建 License
func CreateLicenseViaAPI(apiBase, token, licenseKey, plan string, planConfig PlanConfig) error {
	createURL := fmt.Sprintf("%s/api/content?type=License", apiBase)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fields := map[string]string{
		"id":                "-1",
		"license_key":       licenseKey,
		"plan":              plan,
		"max_devices":       fmt.Sprintf("%d", planConfig.MaxDevices),
		"max_ips":           fmt.Sprintf("%d", planConfig.MaxIPs),
		"sync_enabled":      fmt.Sprintf("%t", planConfig.SyncEnabled),
		"sync_quota":        fmt.Sprintf("%d", planConfig.SyncQuotaMB),
		"publish_enabled":   fmt.Sprintf("%t", planConfig.PublishEnabled),
		"max_sites":         fmt.Sprintf("%d", planConfig.MaxSites),
		"max_storage":       fmt.Sprintf("%d", planConfig.MaxStorageMB),
		"custom_domain":     fmt.Sprintf("%t", planConfig.CustomDomain),
		"custom_sub_domain": fmt.Sprintf("%t", planConfig.CustomSubDomain),
		"validity_days":     fmt.Sprintf("%d", planConfig.ValidityDays),
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", createURL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
