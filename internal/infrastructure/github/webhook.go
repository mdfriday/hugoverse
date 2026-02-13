package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ReleasePayload GitHub Release Webhook 的 payload 结构
type ReleasePayload struct {
	Action     string `json:"action"`
	Release    Release `json:"release"`
	Repository Repository `json:"repository"`
}

// Release GitHub Release 信息
type Release struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// Repository GitHub 仓库信息
type Repository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
}

// VerifySignature 验证 GitHub Webhook 签名
// signature 格式: "sha256=<hex>"
func VerifySignature(payload []byte, signature string, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	// 提取签名部分
	expectedSig := strings.TrimPrefix(signature, "sha256=")

	// 使用 secret 计算 HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	actualSig := hex.EncodeToString(mac.Sum(nil))

	// 常量时间比较，防止时序攻击
	return hmac.Equal([]byte(expectedSig), []byte(actualSig))
}

// ParseReleasePayload 解析 Release Webhook payload
func ParseReleasePayload(data []byte) (*ReleasePayload, error) {
	var payload ReleasePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse release payload: %w", err)
	}
	return &payload, nil
}

// IsPublishedEvent 判断是否是 published 事件
func (p *ReleasePayload) IsPublishedEvent() bool {
	return p.Action == "published"
}

// IsTargetRepository 判断是否是目标仓库
func (p *ReleasePayload) IsTargetRepository(fullName string) bool {
	return p.Repository.FullName == fullName
}

