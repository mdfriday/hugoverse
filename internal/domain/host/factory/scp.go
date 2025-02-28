package factory

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mdfriday/hugoverse/internal/domain/host/entity"
	"github.com/mdfriday/hugoverse/internal/domain/host/valueobject"
)

func newScpHost(name, host, port, remotePath string) (*valueobject.SCPConfig, error) {
	if port == "" {
		port = "22"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to parse port number %s: %v", port, err))
	}

	scpConfig := &valueobject.SCPConfig{
		Username:   name,
		Hostname:   host,
		Port:       portNum,
		RemotePath: remotePath,
	}
	if err := scpConfig.Validate(); err != nil {
		return nil, err
	}

	return scpConfig, nil
}

func NewPasswordScpHost(name, pwd, host, port, remotePath string) (*entity.Host, error) {
	scpConfig, err := newScpHost(name, host, port, remotePath)
	if err != nil {
		return nil, err
	}

	return &entity.Host{
		Netlify: nil,
		SCPHost: entity.NewSCPHost(scpConfig, valueobject.PasswordAuth{Password: pwd}),
	}, nil
}

func NewKeyScpHost(name, host, port, remotePath string, privateKeyPath, passphrase string) (*entity.Host, error) {
	scpConfig, err := newScpHost(name, host, port, remotePath)
	if err != nil {
		return nil, err
	}

	return &entity.Host{
		Netlify: nil,
		SCPHost: entity.NewSCPHost(scpConfig, valueobject.KeyAuth{PrivateKeyPath: privateKeyPath, Passphrase: passphrase}),
	}, nil
}
