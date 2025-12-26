package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (c *Content) GetIPByAddress(licenseKey, address string) (*valueobject.LicenseIP, error) {
	hashKey := hash.MD5(licenseKey + ":" + address)

	data, err := c.GetContentByHash("LicenseIP", hashKey, "")
	if err != nil {
		return nil, err
	}

	var ip valueobject.LicenseIP
	if err := json.Unmarshal(data, &ip); err != nil {
		return nil, err
	}

	return &ip, nil
}

func (c *Content) UpdateLicenseIP(ip *valueobject.LicenseIP) error {
	return c.UpdateContentObject(ip)
}

func (c *Content) CreateLicenseIP(ip *valueobject.LicenseIP) (string, error) {
	return c.newContent("LicenseIP", ip)
}
