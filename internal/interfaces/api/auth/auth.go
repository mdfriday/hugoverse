package auth

import (
	"net/http"
	"os"

	"github.com/mdfriday/hugoverse/internal/interfaces/api/token"
	"github.com/mdfriday/hugoverse/pkg/hash"
	"github.com/mdfriday/hugoverse/pkg/identity"
)

type Auth struct {
	Session string
	UserId  string
}

func (a *Auth) CheckGetMethod(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			next.ServeHTTP(res, req)
			return
		case http.MethodPost:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		default:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}

func (a *Auth) CheckPostMethod(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		case http.MethodPost:
			next.ServeHTTP(res, req)
			return
		default:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}

func (a *Auth) CheckSignature(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, err := token.GetEmailFromSignature(req)
		if err == nil {
			next.ServeHTTP(res, req)
			return
		}

		res.WriteHeader(http.StatusUnauthorized)
	})
}

// Check is HTTP middleware to ensure the request has proper token credentials
func (a *Auth) Check(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if a.IsValid(req) {
			next.ServeHTTP(res, req)
			return
		}

		res.WriteHeader(http.StatusUnauthorized)
	})
}

func (a *Auth) CheckWithRedirect(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		dockerContainer := os.Getenv("DOCKER_CONTAINER")
		
		// 记录请求信息
		println("=== CheckWithRedirect ===")
		println("  Path:", req.URL.Path)
		println("  Host:", req.Host)
		println("  Scheme:", req.URL.Scheme)
		println("  X-Forwarded-Proto:", req.Header.Get("X-Forwarded-Proto"))
		println("  TLS:", req.TLS != nil)
		println("  DOCKER_CONTAINER:", dockerContainer)
		
		// 确定使用的协议
		scheme := "https"
		
		// 1. 优先使用 X-Forwarded-Proto header（来自反向代理）
		if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
			println("  -> Using X-Forwarded-Proto:", scheme)
		} else if req.TLS != nil {
			// 2. 如果是 TLS 连接，使用 https
			scheme = "https"
			println("  -> Using TLS: https")
		} else if req.URL.Scheme != "" {
			// 3. 使用请求 URL 中的 scheme
			scheme = req.URL.Scheme
			println("  -> Using URL.Scheme:", scheme)
		} else {
			// 4. 默认 HTTP（非 TLS 连接）
			scheme = "http"
			println("  -> Default: http")
		}
		
		// 在 Docker 环境中，强制使用 HTTP（本地测试）
		if dockerContainer == "true" {
			scheme = "http"
			println("  -> Docker override: http")
		}
		
		// 构建重定向 URL
		host := req.Host
		if host == "" {
			host = req.URL.Host
		}
		redirect := scheme + "://" + host + "/admin/login"
		
		println("  Final redirect:", redirect)
		println("  IsValid:", a.IsValid(req))
		println("=========================")

		if a.IsValid(req) {
			next.ServeHTTP(res, req)
		} else {
			http.Redirect(res, req, redirect, http.StatusFound)
		}
	})
}

// IsValid checks if the user request is authenticated
func (a *Auth) IsValid(req *http.Request) bool {
	_, err := token.GetToken(req)
	if err != nil {
		return false
	}

	email, _ := token.GetEmail(req)
	a.Session = identity.GenerateSessionID()
	a.UserId = hash.MD5(email)
	return true
}
