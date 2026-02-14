package licensekit

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// ValidateEmail 验证邮箱格式是否合法
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	// RFC 5322 标准的邮箱格式验证
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// CreateUser 创建用户
// apiBase: API 基础 URL，例如 http://127.0.0.1:1314
// email: 用户邮箱
// password: 用户密码
func CreateUser(apiBase, email, password string) error {
	// 验证邮箱格式
	if !ValidateEmail(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}

	userURL := fmt.Sprintf("%s/api/user", apiBase)

	// 构造表单数据
	data := fmt.Sprintf("email=%s&password=%s", email, password)

	req, err := http.NewRequest("POST", userURL, strings.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

	// 接受 200 OK 或 201 Created
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

