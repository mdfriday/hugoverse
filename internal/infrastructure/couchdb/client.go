package couchdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	URL       string `json:"url"`        // CouchDB 服务器地址 (如 http://localhost:5984)
	AdminUser string `json:"admin_user"` // 管理员用户名
	AdminPass string `json:"admin_pass"` // 管理员密码
	DBPrefix  string `json:"db_prefix"`  // 用户数据库前缀 (如 userdb-)
}

// Client CouchDB HTTP 客户端
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient 创建 CouchDB 客户端
func NewClient(config *Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateDatabase 创建数据库
func (c *Client) CreateDatabase(name string) error {
	url := fmt.Sprintf("%s/%s", c.config.URL, name)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer resp.Body.Close()

	// 201 Created 或 412 Already exists 都算成功
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusPreconditionFailed {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create database (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateUser 创建 CouchDB 用户
func (c *Client) CreateUser(email, password string) error {
	userDoc := map[string]interface{}{
		"_id":      fmt.Sprintf("org.couchdb.user:%s", email),
		"name":     email,
		"password": password,
		"roles":    []string{},
		"type":     "user",
	}

	body, err := json.Marshal(userDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal user doc: %w", err)
	}

	url := fmt.Sprintf("%s/_users/org.couchdb.user:%s", c.config.URL, email)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

	// 201 Created 或 409 Conflict (已存在) 都算成功
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create user (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetDatabasePermission 设置数据库权限
func (c *Client) SetDatabasePermission(dbName, email string) error {
	securityDoc := map[string]interface{}{
		"admins": map[string]interface{}{
			"names": []string{},
			"roles": []string{},
		},
		"members": map[string]interface{}{
			"names": []string{email},
			"roles": []string{},
		},
	}

	body, err := json.Marshal(securityDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal security doc: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_security", c.config.URL, dbName)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set permission: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set permission (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteDatabase 删除数据库 (可选)
func (c *Client) DeleteDatabase(name string) error {
	url := fmt.Sprintf("%s/%s", c.config.URL, name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete database (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Ping 检查 CouchDB 连接
func (c *Client) Ping() error {
	req, err := http.NewRequest(http.MethodGet, c.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setBasicAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to CouchDB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CouchDB returned status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) setBasicAuth(req *http.Request) {
	req.SetBasicAuth(c.config.AdminUser, c.config.AdminPass)
}
