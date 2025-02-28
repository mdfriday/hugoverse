package entity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/mdfriday/hugoverse/internal/domain/host"

	oapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/netlify/open-api/v2/go/models"
	netlify "github.com/netlify/open-api/v2/go/porcelain"
	ooapicontext "github.com/netlify/open-api/v2/go/porcelain/context"
	"github.com/sirupsen/logrus"
)

type Netlify struct {
	client       *netlify.Netlify
	clientLogger *logrus.Logger
	config       *valueobject.NetlifyConfig
	log          loggers.Logger
}

func NewNetlify() (*Netlify, error) {
	formats := strfmt.NewFormats()
	client := netlify.NewHTTPClient(formats)

	logger := logrus.New()
	if err := setupLogging(logger); err != nil {
		logger.Fatal(err)
		return nil, err
	}

	return &Netlify{
		client:       client,
		clientLogger: logger,
		log:          loggers.NewDefault(),
	}, nil
}

// NewNetlifyWithConfig creates a new Netlify instance with the provided configuration
func NewNetlifyWithConfig(config *valueobject.NetlifyConfig) (*Netlify, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	formats := strfmt.NewFormats()
	client := netlify.NewHTTPClient(formats)

	logger := logrus.New()
	if err := setupLogging(logger); err != nil {
		logger.Fatal(err)
		return nil, err
	}

	return &Netlify{
		client:       client,
		clientLogger: logger,
		config:       config,
		log:          loggers.NewDefault(),
	}, nil
}

func (a *Netlify) DeployNewNetlifySite(token string, target string, siteName string, domain string) (valueobject.DeployResult, error) {
	c := &valueobject.NetlifyConfig{
		AuthToken:     token,
		SiteID:        "",
		SiteName:      siteName,
		FullDomain:    domain,
		Directory:     path.Join(target, "public"),
		Draft:         false,
		DeployMessage: "Deployed by MDFriday",
	}

	return a.deploy(c)
}

func (a *Netlify) DeployExistingNetlifySite(token string, target string, siteID string) (valueobject.DeployResult, error) {
	c := &valueobject.NetlifyConfig{
		AuthToken:     token,
		SiteID:        siteID,
		Directory:     path.Join(target, "public"),
		Draft:         false,
		DeployMessage: "Deployed by MDFriday",
	}

	return a.deploy(c)
}

// Deploy implements the Deployer interface
func (a *Netlify) Deploy(localPath string) (host.Result, error) {
	result := valueobject.DeployResult{}

	if a.config == nil {
		return result, errors.New("netlify configuration is not set")
	}

	// Check if the directory exists
	info, err := os.Stat(localPath)
	if err != nil {
		return result, fmt.Errorf("failed to access local path: %w", err)
	}

	if !info.IsDir() {
		return result, errors.New("local path must be a directory")
	}

	// If directory is not set in config, use the provided localPath
	// Otherwise, use the configured directory
	deployDir := a.config.Directory
	if deployDir == "" {
		// By default, Netlify expects the "public" directory
		deployDir = filepath.Join(localPath, "public")

		// Check if public directory exists
		if _, err := os.Stat(deployDir); os.IsNotExist(err) {
			// If public directory doesn't exist, use the provided path directly
			deployDir = localPath
		}
	}

	// Create a copy of the config with the updated directory
	deployConfig := *a.config
	deployConfig.Directory = deployDir

	return a.deploy(&deployConfig)
}

func (a *Netlify) deploy(c *valueobject.NetlifyConfig) (valueobject.DeployResult, error) {
	result := valueobject.DeployResult{}
	info, err := os.Stat(c.Directory)

	if os.IsNotExist(err) {
		return result, errors.New("file not exist")
	}

	if !info.IsDir() {
		return result, errors.New("target is not a directory")
	}

	ctx := setupContext(c, a.clientLogger)

	siteID := c.SiteID
	if siteID == "" {
		// 创建新 Netlify 站点
		newSite, err := a.client.CreateSite(ctx, &models.SiteSetup{
			Site: models.Site{
				//AccountSlug:  "admin-zbpioce",
				Name:         c.SiteName,
				CustomDomain: c.FullDomain,
				Ssl:          true,
			},
			SiteSetupAllOf1: models.SiteSetupAllOf1{},
		}, true)
		if err != nil {
			a.log.Errorf("failed to create Netlify site: %s", err)
			return result, err
		}

		// 更新 SiteID
		siteID = newSite.ID
		a.log.Println("Created new site with ID: " + siteID)
	}

	// Deploy site
	resp, err := a.client.DoDeploy(ctx, &netlify.DeployOptions{
		SiteID:  siteID,
		Dir:     c.Directory,
		IsDraft: c.Draft,
		Title:   c.DeployMessage,
	}, nil)
	if err != nil {
		a.log.Errorf("failed to deploy site: %s", err)
		return result, err
	}

	result.SiteID = siteID

	// Set the deployment URL
	if resp.DeploySslURL != "" {
		result.URL = resp.DeploySslURL
	} else {
		result.URL = resp.DeployURL
	}

	result.Message = "Successfully deployed to Netlify"

	// Log the deployment URL
	a.log.Println("Deployed site: " + result.URL)

	return result, nil
}

func (a *Netlify) DeleteNetlifySite(token string, siteID string) error {
	c := &valueobject.NetlifyConfig{
		AuthToken: token,
		SiteID:    siteID,
	}

	ctx := setupContext(c, a.clientLogger)

	a.log.Println("delete site from Netlify...", siteID)

	return a.client.DeleteSite(ctx, c.SiteID)
}

func setupLogging(logger *logrus.Logger) error {
	logLevel, err := logrus.ParseLevel("debug")
	if err != nil {
		return fmt.Errorf("failed to parse log level: %s", err)
	}

	logger.SetLevel(logLevel)
	logger.SetFormatter(&logrus.TextFormatter{})

	return nil
}

func setupContext(c *valueobject.NetlifyConfig, logger *logrus.Logger) ooapicontext.Context {
	ctx := ooapicontext.WithLogger(context.Background(), logger.WithFields(logrus.Fields{
		"source": "netlify",
	}))
	return ooapicontext.WithAuthInfo(ctx, oapiclient.BearerToken(c.AuthToken))
}
