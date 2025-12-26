package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// ========== License Operations ==========

// GetLicenseByKey 通过 LicenseKey 获取 License
func (c *Content) GetLicenseByKey(licenseKey string) (*valueobject.License, error) {
	hashKey := hash.MD5(licenseKey)

	data, err := c.GetContentByHash("License", hashKey, "")
	if err != nil {
		return nil, err
	}

	var license valueobject.License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, err
	}

	return &license, nil
}

// UpdateLicense 更新 License
func (c *Content) UpdateLicense(license *valueobject.License) error {
	return c.UpdateContentObject(license)
}
