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

// GetLicenseCount 获取 License 总数
// 用于判断是否已经生成过企业 License
func (c *Content) GetLicenseCount() int {
	allLicenses := c.AllContents("License")
	return len(allLicenses)
}

// HasAnyLicense 检查是否存在任何 License
// 用于判断企业功能是否已配置
func (c *Content) HasAnyLicense() bool {
	return c.GetLicenseCount() > 0
}

// UpdateLicense 更新 License
func (c *Content) UpdateLicense(license *valueobject.License) error {
	return c.UpdateContentObject(license)
}

// CreateLicense 创建 License
func (c *Content) CreateLicense(license *valueobject.License) (string, error) {
	return c.newContent("License", license)
}

// ========== LicenseUsage Operations ==========

// GetLicenseUsageByKey 通过 LicenseKey 获取 LicenseUsage
func (c *Content) GetLicenseUsageByKey(licenseKey string) (*valueobject.LicenseUsage, error) {
	hashKey := hash.MD5(licenseKey)

	data, err := c.GetContentByHash("LicenseUsage", hashKey, "")
	if err != nil {
		return nil, err
	}

	var licenseUsage valueobject.LicenseUsage
	if err := json.Unmarshal(data, &licenseUsage); err != nil {
		return nil, err
	}

	return &licenseUsage, nil
}

// UpdateLicenseUsage 更新 LicenseUsage
func (c *Content) UpdateLicenseUsage(licenseUsage *valueobject.LicenseUsage) error {
	return c.UpdateContentObject(licenseUsage)
}

// CreateLicenseUsage 创建 LicenseUsage
func (c *Content) CreateLicenseUsage(licenseUsage *valueobject.LicenseUsage) (string, error) {
	return c.newContent("LicenseUsage", licenseUsage)
}

// ========== LicenseTrial Operations ==========

// GetLicenseTrialByEmail 通过 Email 获取 LicenseTrial
func (c *Content) GetLicenseTrialByEmail(email string) (*valueobject.LicenseTrial, error) {
	hashKey := hash.MD5(email)

	data, err := c.GetContentByHash("LicenseTrial", hashKey, "")
	if err != nil {
		return nil, err
	}

	var licenseTrial valueobject.LicenseTrial
	if err := json.Unmarshal(data, &licenseTrial); err != nil {
		return nil, err
	}

	return &licenseTrial, nil
}

// CreateLicenseTrial 创建 LicenseTrial
func (c *Content) CreateLicenseTrial(licenseTrial *valueobject.LicenseTrial) (string, error) {
	return c.newContent("LicenseTrial", licenseTrial)
}
