package entity

import (
	"encoding/json"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/hash"
)

func (c *Content) GetSubDomainByKey(sub string) (*valueobject.SubDomain, error) {
	hashKey := hash.MD5(sub)

	data, err := c.GetContentByHash("SubDomain", hashKey, "")
	if err != nil {
		return nil, err
	}

	var sd valueobject.SubDomain
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}

	return &sd, nil
}

func (c *Content) UpdateSubDomain(sd *valueobject.SubDomain) error {
	return c.UpdateContentObject(sd)
}

func (c *Content) CreateSubDomain(sd *valueobject.SubDomain) (string, error) {
	return c.newContent("SubDomain", sd)
}

func (c *Content) DeleteSubDomain(sd *valueobject.SubDomain) error {
	return c.DeleteContentObject(sd)
}
