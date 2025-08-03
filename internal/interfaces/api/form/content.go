package form

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	MaxMemory = 8 << 20
)

type Content struct {
}

// Handle 处理请求并检查 Content-Type
func (c *Content) Handle(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// 检查是否是 POST 请求
		if req.Method == http.MethodPost {
			contentType := req.Header.Get("Content-Type")

			// 支持 application/x-www-form-urlencoded
			if contentType == "application/x-www-form-urlencoded" {
				if err := req.ParseForm(); err != nil {
					http.Error(res, "Failed to parse form data", http.StatusBadRequest)
					return
				}
			} else if strings.HasPrefix(contentType, "multipart/form-data") {
				// 支持 multipart/form-data
				if err := req.ParseMultipartForm(MaxMemory); err != nil {
					fmt.Println("[content] Error parsing multipart form data:", err)
					http.Error(res, "Failed to parse multipart form data", http.StatusBadRequest)
					return
				}
			} else {
				http.Error(res, fmt.Sprintf("Unsupported Content-Type: %s. Only application/x-www-form-urlencoded and multipart/form-data are allowed", contentType), http.StatusUnsupportedMediaType)
				return
			}
		}

		// 继续处理请求
		next.ServeHTTP(res, req)
	})
}
