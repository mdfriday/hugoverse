package entity

import (
	"encoding/json"
	"fmt"
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

// GetIPsByLicense 获取某个 License 的所有 IP 记录
// 使用前缀查询: slug 格式为 "{license}:{ipAddress}"
func (c *Content) GetIPsByLicense(licenseKey string) ([]valueobject.LicenseIP, error) {
	// 使用 slug 前缀查询
	prefix := fmt.Sprintf("%s:", licenseKey)
	
	results, err := c.Repo.ContentByPrefix("LicenseIP", prefix)
	if err != nil {
		return nil, err
	}
	
	ips := make([]valueobject.LicenseIP, 0, len(results))
	for _, data := range results {
		var ip valueobject.LicenseIP
		if err := json.Unmarshal(data, &ip); err != nil {
			c.Log.Errorf("Failed to unmarshal IP: %v", err)
			continue
		}
		ips = append(ips, ip)
	}
	
	return ips, nil
}

func (c *Content) UpdateLicenseIP(ip *valueobject.LicenseIP) error {
	return c.UpdateContentObject(ip)
}

func (c *Content) CreateLicenseIP(ip *valueobject.LicenseIP) (string, error) {
	return c.newContent("LicenseIP", ip)
}
