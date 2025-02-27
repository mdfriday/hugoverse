package entity

import (
	"context"
	"errors"
	"fmt"
	oapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/netlify/open-api/v2/go/models"
	netlify "github.com/netlify/open-api/v2/go/porcelain"
	ooapicontext "github.com/netlify/open-api/v2/go/porcelain/context"
	"github.com/sirupsen/logrus"
	"os"
	"path"
)

type Netlify struct {
	client       *netlify.Netlify
	clientLogger *logrus.Logger

	log loggers.Logger
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

		log: loggers.NewDefault(),
	}, nil
}

func (a *Netlify) DeployNewNetlifySite(token string, target string, siteName string, domain string) (string, error) {
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

func (a *Netlify) DeployExistingNetlifySite(token string, target string, siteID string) (string, error) {
	c := &valueobject.NetlifyConfig{
		AuthToken:     token,
		SiteID:        siteID,
		Directory:     path.Join(target, "public"),
		Draft:         false,
		DeployMessage: "Deployed by MDFriday",
	}

	return a.deploy(c)
}

func (a *Netlify) deploy(c *valueobject.NetlifyConfig) (string, error) {
	info, err := os.Stat(c.Directory)

	if os.IsNotExist(err) {
		return "", errors.New("file not exist")
	}

	if !info.IsDir() {
		return "", errors.New("target is not a directory")
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
		}, true) // 设置 configureDNS 为 true
		if err != nil {
			a.log.Errorf("failed to create Netlify site: %s", err)
			return "", err
		}

		// 更新 SiteID
		siteID = newSite.ID
		a.log.Println("Created new site with ID: " + c.SiteID)
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
		return "", err
	}

	// Print the site URL
	if resp.DeploySslURL != "" {
		a.log.Println("Deployed site: " + resp.DeploySslURL)
	} else if resp.DeployURL != "" {
		a.log.Println("Deployed site: " + resp.DeployURL)
	}

	return siteID, nil
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
