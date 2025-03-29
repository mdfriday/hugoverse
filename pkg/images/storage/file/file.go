package file

import (
	"context"
	"errors"
	"github.com/mdfriday/hugoverse/pkg/images/storage"
	"os"
)

// Provider implements a file-based image storage
type Provider struct {
	path string
}

// New returns a new Provider instance
func New(path string) (*Provider, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &Provider{
		path,
	}, nil
}

// Get returns the image data for an image id
func (p *Provider) Get(ctx context.Context, id string) ([]byte, error) {
	imageData, err := os.ReadFile(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrNotFound
		}

		return nil, err
	}

	return imageData, nil
}
