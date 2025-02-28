package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/host/entity"
	"github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
)

// NewNetlifyHost creates a basic Netlify host without configuration
// This is kept for backward compatibility
func NewNetlifyHost() (*entity.Host, error) {
	netlify, err := entity.NewNetlify()
	if err != nil {
		return nil, err
	}

	return &entity.Host{
		Netlify: netlify,
	}, nil
}

// NewNetlifyHostForNewSite creates a Netlify host configured for deploying a new site
func NewNetlifyHostForNewSite(authToken, siteName, domain string) (*entity.Host, error) {
	config := &valueobject.NetlifyConfig{
		AuthToken:     authToken,
		SiteID:        "", // Empty for new site
		SiteName:      siteName,
		FullDomain:    domain,
		Directory:     "",
		Draft:         false,
		DeployMessage: "Deployed by MDFriday",
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	netlify, err := entity.NewNetlifyWithConfig(config)
	if err != nil {
		return nil, err
	}

	return &entity.Host{
		Netlify: netlify,
	}, nil
}

// NewNetlifyHostForExistingSite creates a Netlify host configured for deploying to an existing site
func NewNetlifyHostForExistingSite(authToken, siteID string) (*entity.Host, error) {
	config := &valueobject.NetlifyConfig{
		AuthToken:     authToken,
		SiteID:        siteID,
		Directory:     "",
		Draft:         false,
		DeployMessage: "Deployed by MDFriday",
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	netlify, err := entity.NewNetlifyWithConfig(config)
	if err != nil {
		return nil, err
	}

	return &entity.Host{
		Netlify: netlify,
	}, nil
}
