package factory

import (
	"github.com/mdfriday/hugoverse/internal/domain/host"
	"github.com/mdfriday/hugoverse/internal/domain/host/entity"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// NewHost creates a new Host instance with optional SCP configuration
func NewHost(log loggers.Logger, scpConfig *host.SCPConfig) (*entity.Host, error) {
	netlify, err := entity.NewNetlify(log)
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
		scpHost.SetLogger(log)
		h.SCPHost = scpHost
	}

	return h, nil
}
