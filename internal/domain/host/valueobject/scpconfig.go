package valueobject

import (
	"errors"
)

// SCPConfig defines the configuration for SCP deployment
type SCPConfig struct {
	Username   string
	Hostname   string
	Port       int
	RemotePath string
}

func (s *SCPConfig) Validate() error {
	if s.Hostname == "" {
		return errors.New("hostname is required")
	}

	if s.Username == "" {
		return errors.New("username is required")
	}

	if s.RemotePath == "" {
		return errors.New("remote path is required")
	}

	if s.Port == 0 {
		s.Port = 22
	}

	return nil
}
