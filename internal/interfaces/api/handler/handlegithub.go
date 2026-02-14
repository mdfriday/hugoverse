package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/infrastructure/github"
)

// GithubReleaseHander 处理 GitHub Release Webhook
// POST /api/github-hook
func (s *Handler) GithubReleaseHander(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(req.Body)
	if err != nil {
		s.log.Errorf("Failed to read request body: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// 获取 GitHub 配置
	hookSecret := s.adminApp.HookSecret()
	if hookSecret == "" {
		s.log.Errorln("GitHub Hook Secret not configured")
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "GitHub Hook Secret not configured")
		return
	}

	githubToken := s.adminApp.GithubToken()
	if githubToken == "" {
		s.log.Warnln("GitHub Token not configured, may fail for private repositories")
	}

	targetRepo := s.adminApp.TargetRepository()

	// 验证签名
	signature := req.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		s.log.Errorln("Missing X-Hub-Signature-256 header")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !github.VerifySignature(body, signature, hookSecret) {
		s.log.Errorln("Invalid webhook signature")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 解析 payload
	// 根据 GitHub 官方文档: https://docs.github.com/zh/webhooks/webhook-events-and-payloads
	// GitHub Webhook 支持两种 Content-Type：
	// 1. application/json: JSON 直接在 body 中（推荐）
	// 2. application/x-www-form-urlencoded: JSON 在表单字段 "payload" 中
	var payloadData []byte
	contentType := req.Header.Get("Content-Type")
	
	s.log.Infof("Webhook received - Content-Type: %s, Body length: %d", contentType, len(body))
	
	if strings.Contains(contentType, "application/json") {
		// 格式 1: JSON 直接在 body 中
		s.log.Infof("Parsing as application/json")
		payloadData = body
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// 格式 2: JSON 在表单字段 "payload" 中
		s.log.Infof("Parsing as application/x-www-form-urlencoded")
		
		// 需要重新包装 body 供 ParseForm 使用
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		if err := req.ParseForm(); err != nil {
			s.log.Errorf("Failed to parse form: %v", err)
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(res, "Failed to parse form data")
			return
		}
		
		payloadStr := req.FormValue("payload")
		if payloadStr == "" {
			s.log.Errorf("Missing 'payload' field in form data")
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(res, "Missing 'payload' field in form data")
			return
		}
		
		payloadData = []byte(payloadStr)
		s.log.Infof("Extracted payload from form field (length: %d)", len(payloadData))
	} else {
		s.log.Errorf("Unsupported Content-Type: %s", contentType)
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Unsupported Content-Type: %s", contentType)
		return
	}

	s.log.Infof("Parsing JSON payload (length: %d)", len(payloadData))
	payload, err := github.ParseReleasePayload(payloadData)
	if err != nil {
		s.log.Errorf("Failed to parse release payload: %v", err)
		// 显示前 500 个字符用于调试
		maxLen := 500
		if len(payloadData) < maxLen {
			maxLen = len(payloadData)
		}
		s.log.Errorf("Payload data (first %d chars): %s", maxLen, string(payloadData[:maxLen]))
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Failed to parse payload: %v", err)
		return
	}

	// 只处理 published 事件
	if !payload.IsPublishedEvent() {
		s.log.Infof("Ignoring non-published event: %s", payload.Action)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Event %s ignored", payload.Action)
		return
	}

	// 验证是否是目标仓库
	if !payload.IsTargetRepository(targetRepo) {
		s.log.Warnf("Received webhook from unexpected repository: %s (expected: %s)", 
			payload.Repository.FullName, targetRepo)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Repository %s ignored", payload.Repository.FullName)
		return
	}

	s.log.Infof("Received published release event: %s (%s) from %s",
		payload.Release.Name,
		payload.Release.TagName,
		payload.Repository.FullName)

	// 检查是否有 assets
	if len(payload.Release.Assets) == 0 {
		s.log.Warnf("Release %s has no assets, skipping download", payload.Release.TagName)
		res.WriteHeader(http.StatusOK)
		fmt.Fprintf(res, "Release %s has no assets", payload.Release.TagName)
		return
	}

	s.log.Infof("Found %d assets in release %s", len(payload.Release.Assets), payload.Release.TagName)

	// 下载 assets 并打包成 ZIP
	downloader := github.NewReleaseDownloader()
	uploadDir := application.UploadDir()
	targetFilename := "friday-latest.zip"

	s.log.Infof("Downloading %d assets from release %s", len(payload.Release.Assets), payload.Release.TagName)

	if err := downloader.DownloadAssetsAsZip(
		payload.Release.Assets,
		uploadDir,
		targetFilename,
		githubToken,
	); err != nil {
		s.log.Errorf("Failed to download release assets: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Failed to download release assets: %v", err)
		return
	}

	targetPath := filepath.Join(uploadDir, targetFilename)
	s.log.Infof("Successfully downloaded %d assets from release %s to %s",
		len(payload.Release.Assets),
		payload.Release.TagName,
		targetPath)

	// 返回成功响应
	res.WriteHeader(http.StatusOK)
	fmt.Fprintf(res, "Successfully processed release %s (%s)",
		payload.Release.Name,
		payload.Release.TagName)
}
