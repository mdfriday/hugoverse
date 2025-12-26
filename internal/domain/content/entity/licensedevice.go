package entity

import (
	"encoding/json"
	"fmt"
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

// GetDevicesByLicense 获取某个 License 的所有设备
// 使用前缀查询: slug 格式为 "{license}:{deviceID}"
func (c *Content) GetDevicesByLicense(licenseKey string) ([]valueobject.LicenseDevice, error) {
	// 使用 slug 前缀查询
	prefix := fmt.Sprintf("%s:", licenseKey)
	
	results, err := c.Repo.ContentByPrefix("LicenseDevice", prefix)
	if err != nil {
		return nil, err
	}
	
	devices := make([]valueobject.LicenseDevice, 0, len(results))
	for _, data := range results {
		var device valueobject.LicenseDevice
		if err := json.Unmarshal(data, &device); err != nil {
			c.Log.Errorf("Failed to unmarshal device: %v", err)
			continue
		}
		devices = append(devices, device)
	}
	
	return devices, nil
}

func (c *Content) UpdateDevice(device *valueobject.LicenseDevice) error {
	return c.UpdateContentObject(device)
}

func (c *Content) CreateDevice(device *valueobject.LicenseDevice) (string, error) {
	return c.newContent("LicenseDevice", device)
}
