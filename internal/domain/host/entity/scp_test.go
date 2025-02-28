package entity

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
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
	tempDir, err := os.MkdirTemp("", "scp-test")
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
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	config := &valueobject.SCPConfig{
		Username:   username,
		Hostname:   hostname,
		Port:       22,
		RemotePath: "/tmp/scp-test",
	}
	auth := &valueobject.PasswordAuth{
		Password: password,
	}
	host := NewSCPHost(config, auth)
	host.SetLogger(loggers.NewDefault())

	// Test deployment
	if _, err := host.Deploy(tempDir); err != nil {
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
	config := &valueobject.SCPConfig{
		Username:   username,
		Hostname:   hostname,
		Port:       22,
		RemotePath: "/tmp/scp-test-dir",
	}
	auth := &valueobject.PasswordAuth{
		Password: password,
	}
	host := NewSCPHost(config, auth)
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
	tempDir, err := os.MkdirTemp("", "scp-test-upload")
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
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	config := &valueobject.SCPConfig{
		Username:   username,
		Hostname:   hostname,
		Port:       22,
		RemotePath: "/tmp/scp-test-upload",
	}
	auth := &valueobject.PasswordAuth{
		Password: password,
	}
	host := NewSCPHost(config, auth)
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
	tempDir, err := os.MkdirTemp("", "scp-test-tar")
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
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create SCPHost instance
	config := &valueobject.SCPConfig{
		Username:   username,
		Hostname:   hostname,
		Port:       22,
		RemotePath: "/tmp/scp-test-tar",
	}
	auth := &valueobject.PasswordAuth{
		Password: password,
	}
	host := NewSCPHost(config, auth)
	host.SetLogger(loggers.NewDefault())

	// Test deployment with tar
	if _, err := host.DeployWithTar(tempDir); err != nil {
		t.Fatalf("DeployWithTar failed: %v", err)
	}

	// Verify deployment
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

		// Use cat to verify file content
		output, err := session.Output(fmt.Sprintf("cat %s", remotePath))
		session.Close()

		if err != nil {
			t.Errorf("File %s not found or not readable on remote server: %v", remotePath, err)
			continue
		}

		if string(output) != expectedContent {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", remotePath, expectedContent, string(output))
		}
	}
}

func TestSCPFields_RequiredFields(t *testing.T) {
	// Create SCPHost instance
	config := &valueobject.SCPConfig{
		Username:   "testuser",
		Hostname:   "testhost",
		Port:       22,
		RemotePath: "/tmp/test",
	}
	auth := &valueobject.PasswordAuth{
		Password: "testpass",
	}
	host := NewSCPHost(config, auth)

	// Test that fields are properly set
	fields := host.newSCPFields("test_operation")

	// Check that operation field exists
	found := false
	for _, field := range fields.Fields() {
		if field.Name == "operation" && field.Value == "test_operation" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected operation field with value 'test_operation' not found")
	}
}

func TestSCPHost_SensitiveDataMasking(t *testing.T) {
	// Test username masking
	host := &SCPHost{}

	testCases := []struct {
		input    string
		expected string
	}{
		{"admin", "ad***"},
		{"a", "***"},
		{"", "***"},
		{"root", "ro***"},
		{"verylongusername", "ve***"},
	}

	for _, tc := range testCases {
		result := host.maskUsername(tc.input)
		if result != tc.expected {
			t.Errorf("maskUsername(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestSCPHost_SafeLogPath(t *testing.T) {
	host := &SCPHost{}

	// Save original HOME env var
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set HOME for testing
	os.Setenv("HOME", "/home/testuser")

	testCases := []struct {
		input    string
		expected string
	}{
		{"/home/testuser/file.txt", "~/file.txt"},
		{"/home/testuser", "~"},
		{"/home/testuser/", "~"},
		{"/home/testuser/dir/file.txt", "~/dir/file.txt"},
		{"/var/log/file.txt", "/var/log/file.txt"},
		{"relative/path", "relative/path"},
	}

	for _, tc := range testCases {
		result := host.safeLogPath(tc.input)
		if result != tc.expected {
			t.Errorf("safeLogPath(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}
