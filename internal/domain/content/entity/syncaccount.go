package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (c *Content) GetSyncAccountByLicense(licenseKey string) (*valueobject.SyncAccount, error) {
	hashKey := hash.MD5(licenseKey)

	data, err := c.GetContentByHash("SyncAccount", hashKey, "")
	if err != nil {
		return nil, err
	}

	var sa valueobject.SyncAccount
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, err
	}

	return &sa, nil
}

func (c *Content) UpdateSyncAccount(sa *valueobject.SyncAccount) error {
	return c.UpdateContentObject(sa)
}

func (c *Content) CreateSyncAccount(sa *valueobject.SyncAccount) (string, error) {
	return c.newContent("SyncAccount", sa)
}
