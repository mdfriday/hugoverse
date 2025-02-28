package valueobject

import "fmt"

// DeployResult represents the result of a Netlify deployment
type DeployResult struct {
	SiteID  string
	URL     string
	Message string
}

// GetID implements Result interface
func (r DeployResult) GetID() string {
	return r.SiteID
}

// GetURL implements Result interface
func (r DeployResult) GetURL() string {
	return r.URL
}

// GetMessage implements Result interface
func (r DeployResult) GetMessage() string {
	return r.Message
}

// SCPResult represents the result of an SCP deployment
type SCPResult struct {
	ServerPath string
	HostName   string
	Message    string
}

// GetID implements Result interface
func (r SCPResult) GetID() string {
	return r.HostName
}

// GetURL implements Result interface
func (r SCPResult) GetURL() string {
	return fmt.Sprintf("scp://%s%s", r.HostName, r.ServerPath)
}

// GetMessage implements Result interface
func (r SCPResult) GetMessage() string {
	return r.Message
}
