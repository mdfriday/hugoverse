package valueobject

import "errors"

type NetlifyConfig struct {
	AuthToken     string `envconfig:"auth_token" required:"true"`
	SiteID        string `envconfig:"site_id" required:"true"`
	SiteName      string `default:""`
	FullDomain    string `default:""`
	Directory     string `required:"true"`
	Draft         bool   `default:"true"`
	DeployMessage string `default:""`
}

// Validate ensures that the configuration is valid
func (c *NetlifyConfig) Validate() error {
	if c.AuthToken == "" {
		return errors.New("auth token is required")
	}

	// For new sites, SiteID is empty but SiteName is required
	if c.SiteID == "" && c.SiteName == "" {
		return errors.New("either site ID or site name is required")
	}

	// Set default deploy message if not provided
	if c.DeployMessage == "" {
		c.DeployMessage = "Deployed by MDFriday"
	}

	return nil
}
