package entity

import (
	"errors"
	"github.com/mdfriday/hugoverse/internal/domain/host"
)

type Host struct {
	*Netlify
	*SCPHost
}

// Deploy implements the Deployer interface
func (h *Host) Deploy(localPath string) (host.Result, error) {
	if h.SCPHost != nil {
		return h.SCPHost.Deploy(localPath)
	}

	if h.Netlify != nil {
		return h.Netlify.Deploy(localPath)
	}

	return nil, errors.New("no deployment method available")
}
