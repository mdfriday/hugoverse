package entity

import (
	"encoding/json"

	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

// ========== Instance Operations ==========

// GetInstanceByID 通过 InstanceID 获取 Instance
func (c *Content) GetInstanceByID(instanceID string) (*valueobject.Instance, error) {
	hashKey := hash.MD5(instanceID)

	data, err := c.GetContentByHash("Instance", hashKey, "")
	if err != nil {
		return nil, err
	}

	var instance valueobject.Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return nil, err
	}

	return &instance, nil
}

// CreateInstance 创建 Instance
func (c *Content) CreateInstance(instance *valueobject.Instance) error {
	_, err := c.newContent("Instance", instance)
	return err
}

// UpdateInstance 更新 Instance
func (c *Content) UpdateInstance(instance *valueobject.Instance) error {
	return c.UpdateContentObject(instance)
}

// GetInstanceCount 获取 Instance 总数
func (c *Content) GetInstanceCount() int {
	allInstances := c.AllContents("Instance")
	return len(allInstances)
}
