package entity

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mdfriday/hugoverse/pkg/loggers"

	"github.com/bep/logg"
	"github.com/pkg/errors"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// generateSessionID generates a unique session ID
func generateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// If we can't generate random bytes, use timestamp
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("sess-%s", hex.EncodeToString(b))
}

// AuthMethod represents different authentication methods
type AuthMethod interface {
	SSHAuthMethod() ssh.AuthMethod
}

// PasswordAuth implements password-based authentication
type PasswordAuth struct {
	Password string
}

func (p PasswordAuth) SSHAuthMethod() ssh.AuthMethod {
	return ssh.Password(p.Password)
}

// KeyAuth implements key-based authentication
type KeyAuth struct {
	PrivateKeyPath string
	Passphrase     string
}

func (k KeyAuth) SSHAuthMethod() ssh.AuthMethod {
	key, err := os.ReadFile(k.PrivateKeyPath)
	if err != nil {
		return nil
	}

	var signer ssh.Signer
	if k.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(k.Passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(signer)
}

// SCPHost represents a remote host that supports SCP file transfer
type SCPHost struct {
	Username    string
	Auth        AuthMethod
	Hostname    string
	Port        int
	RemotePath  string
	sshClient   *ssh.Client
	logger      loggers.Logger
	sessionID   string
	HostKeyFile string // Path to known_hosts file
}

// SCPFields implements the logg.Fielder interface for structured logging
type SCPFields struct {
	fields logg.Fields
}

// Fields implements the logg.Fielder interface
func (f *SCPFields) Fields() logg.Fields {
	return f.fields
}

// newSCPFields creates a new SCPFields instance with common fields
func (h *SCPHost) newSCPFields(operation string) *SCPFields {
	return &SCPFields{
		fields: logg.Fields{
			{Name: "sessionID", Value: h.sessionID},
			{Name: "host", Value: h.Hostname},
			{Name: "operation", Value: operation},
		},
	}
}

// addField adds a new field to the SCPFields
func (f *SCPFields) addField(name string, value interface{}) {
	f.fields = append(f.fields, logg.Field{Name: name, Value: value})
}

// addFields adds multiple fields to the SCPFields
func (f *SCPFields) addFields(fields ...logg.Field) {
	f.fields = append(f.fields, fields...)
}

// NewSCPHost creates a new SCPHost instance with password authentication
func NewSCPHost(username, password, hostname string, port int, remotePath string) *SCPHost {
	return &SCPHost{
		Username:    username,
		Auth:        PasswordAuth{Password: password},
		Hostname:    hostname,
		Port:        port,
		RemotePath:  remotePath,
		logger:      loggers.NewDefault(),
		sessionID:   generateSessionID(),
		HostKeyFile: filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"),
	}
}

// NewSCPHostWithKey creates a new SCPHost instance with key authentication
func NewSCPHostWithKey(username, privateKeyPath, hostname string, port int, remotePath string, passphrase string) *SCPHost {
	return &SCPHost{
		Username:    username,
		Auth:        KeyAuth{PrivateKeyPath: privateKeyPath, Passphrase: passphrase},
		Hostname:    hostname,
		Port:        port,
		RemotePath:  remotePath,
		logger:      loggers.NewDefault(),
		sessionID:   generateSessionID(),
		HostKeyFile: filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"),
	}
}

// Connect establishes SSH connection to the remote host
func (h *SCPHost) Connect() error {
	connLog := h.logger.Info()
	defer loggers.TimeTrackf(connLog, time.Now(), nil, "")

	fields := h.newSCPFields("connect")
	fields.addFields(
		logg.Field{Name: "port", Value: h.Port},
		logg.Field{Name: "user", Value: h.Username},
		logg.Field{Name: "authType", Value: fmt.Sprintf("%T", h.Auth)},
	)

	connLog.WithFields(fields).Logf("Establishing SSH connection")

	hostKeyCallback, err := h.getHostKeyCallback()
	if err != nil {
		connLog.WithFields(fields).WithError(err).Logf("Failed to setup host key verification")
		return errors.Wrap(err, "failed to setup host key verification")
	}

	config := &ssh.ClientConfig{
		User: h.Username,
		Auth: []ssh.AuthMethod{
			h.Auth.SSHAuthMethod(),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", h.Hostname, h.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		connLog.WithFields(fields).WithError(err).Logf("Failed to establish SSH connection")
		return errors.Wrapf(err, "failed to connect to %s", addr)
	}

	connLog.WithFields(fields).Logf("Successfully connected to remote host")
	h.sshClient = client
	return nil
}

// getHostKeyCallback returns a callback for host key verification
func (h *SCPHost) getHostKeyCallback() (ssh.HostKeyCallback, error) {
	fields := h.newSCPFields("host_key_verification")
	fields.addField("hostKeyFile", h.HostKeyFile)

	if h.HostKeyFile == "" {
		h.logger.Warn().WithFields(fields).Logf("No host key file specified, using insecure host key verification")
		return ssh.InsecureIgnoreHostKey(), nil
	}

	hostKeyCallback, err := knownhosts.New(h.HostKeyFile)
	if err != nil {
		h.logger.Error().WithFields(fields).WithError(err).Logf("Failed to load known_hosts file")
		return nil, errors.Wrap(err, "failed to load known_hosts file")
	}
	return hostKeyCallback, nil
}

// Close closes the SSH connection
func (h *SCPHost) Close() error {
	if h.sshClient != nil {
		closeLog := h.logger.Info()
		defer loggers.TimeTrackf(closeLog, time.Now(), nil, "")

		fields := h.newSCPFields("close")
		closeLog.WithFields(fields).Logf("Closing SSH connection")

		if err := h.sshClient.Close(); err != nil {
			closeLog.WithFields(fields).WithError(err).Logf("Failed to close SSH connection")
			return errors.Wrap(err, "failed to close SSH connection")
		}

		closeLog.WithFields(fields).Logf("Successfully closed SSH connection")
	}
	return nil
}

// CreateRemoteDirectory creates a directory on the remote host
func (h *SCPHost) CreateRemoteDirectory(path string) error {
	if h.sshClient == nil {
		return errors.New("not connected to remote host")
	}

	dirLog := h.logger.Info()
	defer loggers.TimeTrackf(dirLog, time.Now(), nil, "")

	fields := h.newSCPFields("create_directory")
	fields.addFields(
		logg.Field{Name: "path", Value: path},
	)
	dirLog.WithFields(fields).Logf("Creating remote directory")

	session, err := h.sshClient.NewSession()
	if err != nil {
		dirLog.WithFields(fields).WithError(err).Logf("Failed to create SSH session")
		return errors.Wrap(err, "failed to create session")
	}
	defer session.Close()

	cmd := fmt.Sprintf("mkdir -p %s", path)
	fields.addField("command", cmd)
	dirLog.WithFields(fields).Logf("Executing mkdir command")

	if err := session.Run(cmd); err != nil {
		dirLog.WithFields(fields).WithError(err).Logf("Failed to create remote directory")
		return errors.Wrapf(err, "failed to create remote directory %s", path)
	}

	dirLog.WithFields(fields).Logf("Successfully created remote directory")
	return nil
}

// UploadDirectory recursively uploads a directory to the remote host
func (h *SCPHost) UploadDirectory(localPath string) error {
	if h.sshClient == nil {
		return errors.New("not connected to remote host")
	}

	uploadLog := h.logger.Info()
	defer loggers.TimeTrackf(uploadLog, time.Now(), nil, "")

	fields := h.newSCPFields("upload_directory")
	fields.addFields(
		logg.Field{Name: "localPath", Value: localPath},
		logg.Field{Name: "remotePath", Value: h.RemotePath},
	)
	uploadLog.WithFields(fields).Logf("Starting directory upload")

	// Create base remote directory
	if err := h.CreateRemoteDirectory(h.RemotePath); err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to create base remote directory")
		return errors.Wrap(err, "failed to create base remote directory")
	}

	// Walk through the directory
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			uploadLog.WithFields(fields).WithError(err).Logf("Failed to access path")
			return errors.Wrapf(err, "failed to access path %s", path)
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			uploadLog.WithFields(fields).WithError(err).Logf("Failed to get relative path")
			return errors.Wrapf(err, "failed to get relative path for %s", path)
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Construct remote path
		remotePath := filepath.Join(h.RemotePath, relPath)

		fileFields := h.newSCPFields("process_file")
		fileFields.addFields(
			logg.Field{Name: "path", Value: path},
			logg.Field{Name: "relPath", Value: relPath},
			logg.Field{Name: "size", Value: info.Size()},
			logg.Field{Name: "mode", Value: info.Mode().String()},
		)

		if info.IsDir() {
			uploadLog.WithFields(fileFields).Logf("Creating remote directory")
			return h.CreateRemoteDirectory(remotePath)
		}

		uploadLog.WithFields(fileFields).Logf("Uploading file")
		return h.uploadFileWithPath(path, remotePath)
	})
}

// uploadFileWithPath uploads a single file to the specified remote path
func (h *SCPHost) uploadFileWithPath(localPath, remotePath string) error {
	if h.sshClient == nil {
		return errors.New("not connected to remote host")
	}

	uploadLog := h.logger.Info()
	defer loggers.TimeTrackf(uploadLog, time.Now(), nil, "")

	fields := h.newSCPFields("file_upload")
	fields.addFields(
		logg.Field{Name: "localPath", Value: localPath},
		logg.Field{Name: "remotePath", Value: remotePath},
	)
	uploadLog.WithFields(fields).Logf("Starting file upload")

	session, err := h.sshClient.NewSession()
	if err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to create SSH session")
		return errors.Wrap(err, "failed to create session")
	}
	defer session.Close()

	file, err := os.Open(localPath)
	if err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to open local file")
		return errors.Wrapf(err, "failed to open local file: %s", localPath)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to get file info")
		return errors.Wrapf(err, "failed to get file info: %s", localPath)
	}

	fields.addFields(
		logg.Field{Name: "fileSize", Value: fileInfo.Size()},
		logg.Field{Name: "fileMode", Value: fileInfo.Mode().String()},
	)
	uploadLog.WithFields(fields).Logf("File metadata retrieved")

	// Get file name from path
	fileName := filepath.Base(localPath)
	remoteDir := filepath.Dir(remotePath)

	// Start remote scp process
	w, err := session.StdinPipe()
	if err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to get stdin pipe")
		return errors.Wrap(err, "failed to get stdin pipe")
	}

	if err := session.Start("scp -t " + remoteDir); err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to start scp process")
		return errors.Wrap(err, "failed to start scp")
	}

	// Send file metadata
	fmt.Fprintf(w, "C%04o %d %s\n", fileInfo.Mode().Perm(), fileInfo.Size(), fileName)

	// Copy file content
	written, err := io.Copy(w, file)
	if err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("Failed to copy file content")
		return errors.Wrapf(err, "failed to copy file (wrote %d bytes)", written)
	}

	fields.addField("bytesWritten", written)
	uploadLog.WithFields(fields).Logf("File content copied")

	// Send transfer end signal
	fmt.Fprint(w, "\x00")
	w.Close()

	// Wait for transfer to complete
	if err := session.Wait(); err != nil {
		uploadLog.WithFields(fields).WithError(err).Logf("SCP transfer failed")
		return errors.Wrap(err, "scp transfer failed")
	}

	uploadLog.WithFields(fields).Logf("File upload completed successfully")
	return nil
}

// SetLogger sets the logger for the SCPHost
func (h *SCPHost) SetLogger(logger loggers.Logger) {
	h.logger = logger
}

// Deploy implements the Deployer interface
func (h *SCPHost) Deploy(localPath string) error {
	deployLog := h.logger.Info()
	defer loggers.TimeTrackf(deployLog, time.Now(), nil, "")

	fields := h.newSCPFields("deploy")
	fields.addFields(
		logg.Field{Name: "localPath", Value: localPath},
		logg.Field{Name: "remotePath", Value: h.RemotePath},
	)
	deployLog.WithFields(fields).Logf("Starting SCP deployment")

	if err := h.Connect(); err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to establish connection")
		return errors.Wrap(err, "deployment failed")
	}
	defer h.Close()

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to get file info")
		return errors.Wrapf(err, "failed to get file info for %s", localPath)
	}

	fields.addFields(
		logg.Field{Name: "isDirectory", Value: fileInfo.IsDir()},
		logg.Field{Name: "fileSize", Value: fileInfo.Size()},
	)

	if fileInfo.IsDir() {
		deployLog.WithFields(fields).Logf("Uploading directory")
		if err := h.UploadDirectory(localPath); err != nil {
			deployLog.WithFields(fields).WithError(err).Logf("Failed to upload directory")
			return errors.Wrap(err, "failed to upload directory")
		}
	} else {
		deployLog.WithFields(fields).Logf("Uploading single file")
		if err := h.uploadFileWithPath(localPath, filepath.Join(h.RemotePath, filepath.Base(localPath))); err != nil {
			deployLog.WithFields(fields).WithError(err).Logf("Failed to upload file")
			return errors.Wrap(err, "failed to upload file")
		}
	}

	deployLog.WithFields(fields).Logf("SCP deployment completed successfully")
	return nil
}

// createTarball creates a compressed tar archive of the source directory
func (h *SCPHost) createTarball(sourceDir string) (string, error) {
	tarLog := h.logger.Info()
	defer loggers.TimeTrackf(tarLog, time.Now(), nil, "")

	fields := h.newSCPFields("create_tarball")
	fields.addField("sourceDir", sourceDir)

	tmpFile, err := os.CreateTemp("", "scp-*.tar.gz")
	if err != nil {
		tarLog.WithFields(fields).WithError(err).Logf("Failed to create temp file")
		return "", errors.Wrap(err, "failed to create temp file")
	}
	defer tmpFile.Close()

	tarLog.WithFields(fields).Logf("Creating tarball")

	gw := gzip.NewWriter(tmpFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			tarLog.WithFields(fields).WithError(err).Logf("Failed to access path during tarball creation")
			return errors.Wrapf(err, "failed to access path %s", path)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			tarLog.WithFields(fields).WithError(err).Logf("Failed to get relative path during tarball creation")
			return errors.Wrapf(err, "failed to get relative path for %s", path)
		}

		if relPath == "." {
			return nil
		}

		fileFields := h.newSCPFields("process_file_for_tar")
		fileFields.addFields(
			logg.Field{Name: "path", Value: relPath},
			logg.Field{Name: "size", Value: info.Size()},
			logg.Field{Name: "mode", Value: info.Mode().String()},
		)
		tarLog.WithFields(fileFields).Logf("Processing file for tarball")

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			tarLog.WithFields(fileFields).WithError(err).Logf("Failed to create tar header")
			return errors.Wrapf(err, "failed to create tar header for %s", path)
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			tarLog.WithFields(fileFields).WithError(err).Logf("Failed to write tar header")
			return errors.Wrapf(err, "failed to write tar header for %s", path)
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				tarLog.WithFields(fileFields).WithError(err).Logf("Failed to open file for tarball")
				return errors.Wrapf(err, "failed to open file %s", path)
			}
			defer file.Close()

			written, err := io.Copy(tw, file)
			if err != nil {
				tarLog.WithFields(fileFields).WithError(err).Logf("Failed to write file content to tarball")
				return errors.Wrapf(err, "failed to write file content for %s", path)
			}

			fileFields.addField("bytesWritten", written)
			tarLog.WithFields(fileFields).Logf("File added to tarball")
		}

		return nil
	})

	if err != nil {
		tarLog.WithFields(fields).WithError(err).Logf("Failed to create tarball")
		return "", errors.Wrap(err, "failed to create tarball")
	}

	fields.addField("tarballPath", tmpFile.Name())
	tarLog.WithFields(fields).Logf("Tarball created successfully")
	return tmpFile.Name(), nil
}

// extractTarball extracts the uploaded tarball on the remote server
func (h *SCPHost) extractTarball(remoteTarPath string) error {
	extractLog := h.logger.Info()
	defer loggers.TimeTrackf(extractLog, time.Now(), nil, "")

	fields := h.newSCPFields("extract_tarball")
	fields.addField("remotePath", remoteTarPath)

	extractLog.WithFields(fields).Logf("Extracting tarball on remote server")

	session, err := h.sshClient.NewSession()
	if err != nil {
		extractLog.WithFields(fields).WithError(err).Logf("Failed to create SSH session for extraction")
		return errors.Wrap(err, "failed to create session for extraction")
	}
	defer session.Close()

	cmd := fmt.Sprintf("cd %s && tar xzf %s && rm %s", h.RemotePath, filepath.Base(remoteTarPath), filepath.Base(remoteTarPath))
	fields.addField("command", cmd)
	extractLog.WithFields(fields).Logf("Executing extraction command")

	if err := session.Run(cmd); err != nil {
		extractLog.WithFields(fields).WithError(err).Logf("Failed to extract tarball")
		return errors.Wrapf(err, "failed to extract tarball at %s", remoteTarPath)
	}

	extractLog.WithFields(fields).Logf("Tarball extracted and cleaned up successfully")
	return nil
}

// DeployWithTar implements the Deployer interface using tar compression
func (h *SCPHost) DeployWithTar(localPath string) error {
	deployLog := h.logger.Info()
	defer loggers.TimeTrackf(deployLog, time.Now(), nil, "")

	fields := h.newSCPFields("deploy_with_tar")
	fields.addFields(
		logg.Field{Name: "localPath", Value: localPath},
		logg.Field{Name: "remotePath", Value: h.RemotePath},
	)
	deployLog.WithFields(fields).Logf("Starting SCP deployment with tar")

	if err := h.Connect(); err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to establish connection")
		return errors.Wrap(err, "deployment failed")
	}
	defer h.Close()

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to get file info")
		return errors.Wrapf(err, "failed to get file info for %s", localPath)
	}

	fields.addFields(
		logg.Field{Name: "isDirectory", Value: fileInfo.IsDir()},
		logg.Field{Name: "fileSize", Value: fileInfo.Size()},
	)

	if !fileInfo.IsDir() {
		deployLog.WithFields(fields).Logf("Single file detected, using direct upload")
		return h.Deploy(localPath)
	}

	if err := h.CreateRemoteDirectory(h.RemotePath); err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to create remote directory")
		return errors.Wrap(err, "failed to create remote directory")
	}

	tarPath, err := h.createTarball(localPath)
	if err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to create tarball")
		return errors.Wrap(err, "failed to create tarball")
	}
	defer os.Remove(tarPath)

	fields.addField("tarPath", tarPath)
	deployLog.WithFields(fields).Logf("Uploading tarball")

	remoteTarPath := filepath.Join(h.RemotePath, filepath.Base(tarPath))
	if err := h.uploadFileWithPath(tarPath, remoteTarPath); err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to upload tarball")
		return errors.Wrap(err, "failed to upload tarball")
	}

	if err := h.extractTarball(remoteTarPath); err != nil {
		deployLog.WithFields(fields).WithError(err).Logf("Failed to extract tarball")
		return errors.Wrap(err, "failed to extract tarball")
	}

	deployLog.WithFields(fields).Logf("SCP deployment with tar completed successfully")
	return nil
}
