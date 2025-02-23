package host

// Deployer defines the interface for deploying files
type Deployer interface {
	Deploy(localPath string) error
}

// SCPConfig defines the configuration for SCP deployment
type SCPConfig struct {
	Username     string
	Password     string
	PrivateKey   string // SSH private key content
	Hostname     string
	Port         int
	RemotePath   string
}

// SCPDeployer defines the interface for SCP deployment
type SCPDeployer interface {
	Deployer
	Connect() error
	Close() error
	CreateRemoteDirectory(path string) error
	UploadDirectory(localPath string) error
}
