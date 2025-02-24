package entity

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdfriday/hugoverse/pkg/loggers"
)

func TestSCPHost_Deploy(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Get test configuration from environment
	username := os.Getenv("TEST_SCP_USERNAME")
	password := os.Getenv("TEST_SCP_PASSWORD")
	hostname := os.Getenv("TEST_SCP_HOSTNAME")
	if username == "" || password == "" || hostname == "" {
		t.Fatal("TEST_SCP_USERNAME, TEST_SCP_PASSWORD, and TEST_SCP_HOSTNAME must be set")
	}

	// Create test directory structure
	tempDir, err := ioutil.TempDir("", "scp-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files and directories
	testFiles := map[string]string{
		"file1.txt":           "Test content 1",
		"dir1/file2.txt":      "Test content 2",
		"dir1/dir2/file3.txt": "Test content 3",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	host := NewSCPHost(
		username,
		password,
		hostname,
		22,
		"/tmp/scp-test",
	)
	host.SetLogger(loggers.NewDefault())

	// Test deployment
	if err := host.Deploy(tempDir); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify deployment (requires SSH access to check files)
	if err := host.Connect(); err != nil {
		t.Fatalf("Failed to connect for verification: %v", err)
	}
	defer host.Close()

	// Check if files exist and have correct content
	for path := range testFiles {
		remotePath := filepath.Join("/tmp/scp-test", path)

		// Create new session for each file verification
		session, err := host.sshClient.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session for verification: %v", err)
		}

		cmd := "test -f " + remotePath
		if err := session.Run(cmd); err != nil {
			t.Errorf("File %s not found on remote server: %v", remotePath, err)
		}

		session.Close()
	}
}

func TestSCPHost_CreateRemoteDirectory(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Get test configuration from environment
	username := os.Getenv("TEST_SCP_USERNAME")
	password := os.Getenv("TEST_SCP_PASSWORD")
	hostname := os.Getenv("TEST_SCP_HOSTNAME")
	if username == "" || password == "" || hostname == "" {
		t.Fatal("TEST_SCP_USERNAME, TEST_SCP_PASSWORD, and TEST_SCP_HOSTNAME must be set")
	}

	// Create SCPHost instance
	host := NewSCPHost(
		username,
		password,
		hostname,
		22,
		"/tmp/scp-test-dir",
	)
	host.SetLogger(loggers.NewDefault())

	// Connect to remote host
	if err := host.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer host.Close()

	// Test directory creation
	testPath := "/tmp/scp-test-dir/test1/test2"
	if err := host.CreateRemoteDirectory(testPath); err != nil {
		t.Fatalf("Failed to create remote directory: %v", err)
	}

	// Verify directory exists
	session, err := host.sshClient.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session for verification: %v", err)
	}
	defer session.Close()

	if err := session.Run("test -d " + testPath); err != nil {
		t.Errorf("Directory %s not found on remote server: %v", testPath, err)
	}
}

func TestSCPHost_UploadDirectory(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Get test configuration from environment
	username := os.Getenv("TEST_SCP_USERNAME")
	password := os.Getenv("TEST_SCP_PASSWORD")
	hostname := os.Getenv("TEST_SCP_HOSTNAME")
	if username == "" || password == "" || hostname == "" {
		t.Fatal("TEST_SCP_USERNAME, TEST_SCP_PASSWORD, and TEST_SCP_HOSTNAME must be set")
	}

	// Create test directory structure
	tempDir, err := ioutil.TempDir("", "scp-test-upload")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt":      "Content 1",
		"dir1/file2.txt": "Content 2",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	host := NewSCPHost(
		username,
		password,
		hostname,
		22,
		"/tmp/scp-test-upload",
	)
	host.SetLogger(loggers.NewDefault())

	// Connect and upload
	if err := host.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer host.Close()

	if err := host.UploadDirectory(tempDir); err != nil {
		t.Fatalf("Failed to upload directory: %v", err)
	}

	// Verify uploaded files
	for path := range testFiles {
		remotePath := filepath.Join("/tmp/scp-test-upload", path)

		// Create new session for each file verification
		session, err := host.sshClient.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session for verification: %v", err)
		}

		// Use cat to verify file content
		output, err := session.Output(fmt.Sprintf("cat %s", remotePath))
		session.Close()

		if err != nil {
			t.Errorf("Failed to read file %s: %v", remotePath, err)
			continue
		}

		expectedContent := testFiles[path]
		if string(output) != expectedContent {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", remotePath, expectedContent, string(output))
		}
	}
}

func TestSCPHost_DeployWithTar(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Get test configuration from environment
	username := os.Getenv("TEST_SCP_USERNAME")
	password := os.Getenv("TEST_SCP_PASSWORD")
	hostname := os.Getenv("TEST_SCP_HOSTNAME")
	if username == "" || password == "" || hostname == "" {
		t.Fatal("TEST_SCP_USERNAME, TEST_SCP_PASSWORD, and TEST_SCP_HOSTNAME must be set")
	}

	// Create test directory structure
	tempDir, err := ioutil.TempDir("", "scp-test-tar")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files and directories
	testFiles := map[string]string{
		"file1.txt":           "Test content 1",
		"dir1/file2.txt":      "Test content 2",
		"dir1/dir2/file3.txt": "Test content 3",
		"dir1/dir2/file4.txt": "Test content 4",
		"dir2/file5.txt":      "Test content 5",
		"dir2/file6.txt":      "Test content 6",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	host := NewSCPHost(
		username,
		password,
		hostname,
		22,
		"/tmp/scp-test-tar",
	)

	// Test deployment with tar
	if err := host.DeployWithTar(tempDir); err != nil {
		t.Fatalf("Deploy with tar failed: %v", err)
	}

	// Verify deployment (requires SSH access to check files)
	if err := host.Connect(); err != nil {
		t.Fatalf("Failed to connect for verification: %v", err)
	}
	defer host.Close()

	// Check if files exist and have correct content
	for path, expectedContent := range testFiles {
		remotePath := filepath.Join("/tmp/scp-test-tar", path)

		// Create new session for each file verification
		session, err := host.sshClient.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session for verification: %v", err)
		}

		// Check if file exists
		cmd := fmt.Sprintf("cat %s", remotePath)
		output, err := session.Output(cmd)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", remotePath, err)
		} else if string(output) != expectedContent {
			t.Errorf("File %s content mismatch. Expected: %s, Got: %s", remotePath, expectedContent, string(output))
		}

		session.Close()
	}
}

func TestSCPFields_RequiredFields(t *testing.T) {
	host := NewSCPHost("testuser", "testpass", "localhost", 22, "/tmp")
	fields := host.newSCPFields("test_operation")

	// Check required fields
	requiredFields := map[string]bool{
		"timestamp": false,
		"level":     false,
		"user_id":   false,
		"sessionID": false,
		"host":      false,
		"operation": false,
	}

	for _, field := range fields.Fields() {
		requiredFields[field.Name] = true
	}

	for fieldName, found := range requiredFields {
		if !found {
			t.Errorf("Required field %s not found in SCPFields", fieldName)
		}
	}
}

func TestSCPHost_SensitiveDataMasking(t *testing.T) {
	tests := []struct {
		username string
		want     string
	}{
		{"admin", "ad***"},
		{"a", "***"},
		{"root", "ro***"},
		{"", "***"},
	}

	host := NewSCPHost("testuser", "testpass", "localhost", 22, "/tmp")

	for _, tt := range tests {
		got := host.maskUsername(tt.username)
		if got != tt.want {
			t.Errorf("maskUsername(%q) = %q, want %q", tt.username, got, tt.want)
		}
	}
}

func TestSCPHost_SafeLogPath(t *testing.T) {
	// Save original HOME env
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set test HOME
	testHome := "/home/testuser"
	os.Setenv("HOME", testHome)

	host := NewSCPHost("testuser", "testpass", "localhost", 22, "/tmp")

	tests := []struct {
		path string
		want string
	}{
		{"/home/testuser/secret/file.txt", "~/secret/file.txt"},
		{"/var/log/file.txt", "/var/log/file.txt"},
		{"/home/testuser", "~"},
		{"/home/otheruser/file.txt", "/home/otheruser/file.txt"},
	}

	for _, tt := range tests {
		got := host.safeLogPath(tt.path)
		if got != tt.want {
			t.Errorf("safeLogPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
