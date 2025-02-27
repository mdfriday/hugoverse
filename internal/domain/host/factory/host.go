package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/host"
	"github.com/mdfriday/hugoverse/internal/domain/host/entity"
)

// NewHost creates a new Host instance with optional SCP configuration
func NewHost(scpConfig *host.SCPConfig) (*entity.Host, error) {
	netlify, err := entity.NewNetlify()
	if err != nil {
		return nil, err
	}

	h := &entity.Host{
		Netlify: netlify,
	}

	// If SCP config is provided, initialize SCPHost
	if scpConfig != nil {
		scpHost := entity.NewSCPHost(
			scpConfig.Username,
			scpConfig.Password,
			scpConfig.Hostname,
			scpConfig.Port,
			scpConfig.RemotePath,
		)
		h.SCPHost = scpHost
	}

	return h, nil
}
