package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (c *Content) GetDeviceByID(licenseKey, deviceID string) (*valueobject.LicenseDevice, error) {
	hashKey := hash.MD5(licenseKey + ":" + deviceID)

	data, err := c.GetContentByHash("LicenseDevice", hashKey, "")
	if err != nil {
		return nil, err
	}

	var device valueobject.LicenseDevice
	if err := json.Unmarshal(data, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

func (c *Content) UpdateDevice(device *valueobject.LicenseDevice) error {
	return c.UpdateContentObject(device)
}

func (c *Content) CreateDevice(device *valueobject.LicenseDevice) (string, error) {
	return c.newContent("LicenseDevice", device)
}
