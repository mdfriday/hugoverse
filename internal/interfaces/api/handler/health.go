package handler

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/pkg/version"
	"net/http"
	"os"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status      string `json:"status"`
	Docker      bool   `json:"docker"`
	Initialized bool   `json:"initialized"`
	Version     string `json:"version"`
}

// HealthHandler 健康检查处理器
func (s *Handler) HealthHandler(res http.ResponseWriter, req *http.Request) {
	health := HealthResponse{
		Status:      "healthy",
		Docker:      isDockerEnvironment(),
		Initialized: s.db.SystemInitComplete(),
		Version:     version.CurrentVersion.String(),
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(health)
}

// isDockerEnvironment 检测是否在 Docker 环境中
func isDockerEnvironment() bool {
	// 方法1: 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 方法2: 检查环境变量
	if os.Getenv("DOCKER_CONTAINER") == "true" {
		return true
	}

	return false
}
