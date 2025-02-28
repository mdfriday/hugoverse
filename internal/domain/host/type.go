package host

import "golang.org/x/crypto/ssh"

// Result is the interface that wraps the basic deployment result methods
type Result interface {
	// GetID returns the unique identifier of the deployment
	GetID() string
	// GetURL returns the URL where the deployment can be accessed
	GetURL() string
	// GetMessage returns any additional information about the deployment
	GetMessage() string
}

// Deployer is the interface that wraps the basic Deploy method
type Deployer interface {
	// Deploy deploys the content from localPath and returns a Result
	Deploy(localPath string) (Result, error)
}

// AuthMethod represents different authentication methods
type AuthMethod interface {
	SSHAuthMethod() ssh.AuthMethod
}

// SCPDeployer defines the interface for SCP deployment
type SCPDeployer interface {
	Deployer
	Connect() error
	Close() error
	CreateRemoteDirectory(path string) error
	UploadDirectory(localPath string) error
}
