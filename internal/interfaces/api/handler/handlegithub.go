package handler

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

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
	// GitHub 使用 application/x-www-form-urlencoded 时，payload 在 form 的 payload 字段中
	if err := req.ParseForm(); err != nil {
		s.log.Errorf("Failed to parse form: %v", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	payloadJSON := req.PostForm.Get("payload")
	if payloadJSON == "" {
		s.log.Errorln("Missing payload in form data")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	payload, err := github.ParseReleasePayload([]byte(payloadJSON))
	if err != nil {
		s.log.Errorf("Failed to parse release payload: %v", err)
		res.WriteHeader(http.StatusBadRequest)
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

	// 下载 zipball
	downloader := github.NewReleaseDownloader()
	uploadDir := application.UploadDir()
	targetFilename := "friday-latest.zip"

	s.log.Infof("Downloading zipball from: %s", payload.Release.ZipballURL)

	if err := downloader.DownloadReleaseZip(
		payload.Release.ZipballURL,
		uploadDir,
		targetFilename,
		githubToken,
	); err != nil {
		s.log.Errorf("Failed to download release zip: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Failed to download release: %v", err)
		return
	}

	targetPath := filepath.Join(uploadDir, targetFilename)
	s.log.Infof("Successfully downloaded release %s to %s",
		payload.Release.TagName,
		targetPath)

	// 返回成功响应
	res.WriteHeader(http.StatusOK)
	fmt.Fprintf(res, "Successfully processed release %s (%s)",
		payload.Release.Name,
		payload.Release.TagName)
}
