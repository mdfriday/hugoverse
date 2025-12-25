package couchdb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminVO "github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
)

func TestNewClient(t *testing.T) {
	config := adminVO.DefaultCouchDBConfig()
	client := NewClient(config)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.config != config {
		t.Error("Config not set correctly")
	}

	if client.httpClient == nil {
		t.Error("httpClient not initialized")
	}
}

func TestCreateDatabase(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		if r.URL.Path != "/testdb" {
			t.Errorf("Expected path /testdb, got %s", r.URL.Path)
		}

		// Check basic auth
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Basic auth not provided")
		}
		if username != "admin" || password != "password" {
			t.Error("Wrong credentials")
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
		DBPrefix:  "userdb-",
	}

	client := NewClient(config)
	err := client.CreateDatabase("testdb")

	if err != nil {
		t.Errorf("CreateDatabase failed: %v", err)
	}
}

func TestCreateDatabaseAlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 412 Precondition Failed (database exists)
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.CreateDatabase("existingdb")

	// Should not return error for existing database
	if err != nil {
		t.Errorf("CreateDatabase should not fail for existing db: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		expectedPath := "/_users/org.couchdb.user:test@example.com"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.CreateUser("test@example.com", "userpassword")

	if err != nil {
		t.Errorf("CreateUser failed: %v", err)
	}
}

func TestCreateUserAlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 409 Conflict (user exists)
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.CreateUser("existing@example.com", "password")

	// Should not return error for existing user
	if err != nil {
		t.Errorf("CreateUser should not fail for existing user: %v", err)
	}
}

func TestSetDatabasePermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		expectedPath := "/testdb/_security"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.SetDatabasePermission("testdb", "user@example.com")

	if err != nil {
		t.Errorf("SetDatabasePermission failed: %v", err)
	}
}

func TestGetDatabaseInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"doc_count": 100, "disk_size": 1048576}`))
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	info, err := client.GetDatabaseInfo("testdb")

	if err != nil {
		t.Errorf("GetDatabaseInfo failed: %v", err)
	}

	if info.DocCount != 100 {
		t.Errorf("Expected DocCount 100, got %d", info.DocCount)
	}

	if info.DiskSize != 1048576 {
		t.Errorf("Expected DiskSize 1048576, got %d", info.DiskSize)
	}
}

func TestGetDatabaseInfoNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	_, err := client.GetDatabaseInfo("nonexistent")

	if err == nil {
		t.Error("GetDatabaseInfo should fail for non-existent database")
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/" {
			t.Errorf("Expected path /, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.Ping()

	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestDeleteDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &adminVO.CouchDBConfig{
		URL:       server.URL,
		AdminUser: "admin",
		AdminPass: "password",
	}

	client := NewClient(config)
	err := client.DeleteDatabase("testdb")

	if err != nil {
		t.Errorf("DeleteDatabase failed: %v", err)
	}
}

// Test interface implementation
func TestClientImplementsCouchDBClient(t *testing.T) {
	config := adminVO.DefaultCouchDBConfig()
	client := NewClient(config)

	// This will fail at compile time if Client doesn't implement CouchDBClient
	var _ interface {
		CreateDatabase(name string) error
		CreateUser(email, password string) error
		SetDatabasePermission(dbName, email string) error
	} = client
}

