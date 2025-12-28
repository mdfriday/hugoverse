package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (c *Content) GetPublishDomainByKey(license string) (*valueobject.PublishDomain, error) {
	hashKey := hash.MD5(license)

	data, err := c.GetContentByHash("PublishDomain", hashKey, "")
	if err != nil {
		return nil, err
	}

	var sd valueobject.PublishDomain
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}

	return &sd, nil
}

func (c *Content) UpdatePublishDomain(sd *valueobject.PublishDomain) error {
	return c.UpdateContentObject(sd)
}

func (c *Content) CreatePublishDomain(sd *valueobject.PublishDomain) (string, error) {
	return c.newContent("PublishDomain", sd)
}
