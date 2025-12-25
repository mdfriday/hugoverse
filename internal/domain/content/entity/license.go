package entity

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// ========== License Operations ==========

// GetLicenseByKey 通过 LicenseKey 获取 License
func (c *Content) GetLicenseByKey(licenseKey string) (*valueobject.License, error) {
	hashKey := hash.MD5(licenseKey)

	idBytes, err := c.Repo.GetIdByHash("License", hashKey)
	if err != nil || idBytes == nil {
		return nil, fmt.Errorf("license not found: %s", licenseKey)
	}

	data, err := c.Repo.GetContent("License", string(idBytes))
	if err != nil {
		return nil, err
	}

	var license valueobject.License
	if err := json.Unmarshal(data, &license); err != nil {
		return nil, err
	}

	return &license, nil
}

// CreateLicense 创建新的 License
func (c *Content) CreateLicense(license *valueobject.License) error {
	// 获取新 ID
	id, err := c.Repo.NextContentId("License")
	if err != nil {
		return fmt.Errorf("failed to get next ID: %w", err)
	}
	license.ID = int(id)

	// 初始化 Item 字段 (必须初始化所有必要字段)
	license.Namespace = "License"
	license.Status = "public"
	
	// 初始化 UUID (如果为空)
	if license.UUID.String() == "00000000-0000-0000-0000-000000000000" {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("failed to generate UUID: %w", err)
		}
		license.UUID = newUUID
	}
	
	// 初始化时间戳
	if license.Timestamp == 0 {
		license.Timestamp = license.IssueDate
	}
	if license.Updated == 0 {
		license.Updated = license.IssueDate
	}

	license.SetHash()
	license.SetSlug("")  // 使用默认值 (LicenseKey)

	data, err := json.Marshal(license)
	if err != nil {
		return err
	}
	err = c.Repo.NewContent(license, data)
	if err != nil {
		return err
	}
	return nil
}

// UpdateLicense 更新 License
func (c *Content) UpdateLicense(license *valueobject.License) error {
	data, err := json.Marshal(license)
	if err != nil {
		return err
	}
	return c.Repo.PutContent(license, data)
}

