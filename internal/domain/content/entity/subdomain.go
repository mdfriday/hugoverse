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
	slug, err := valueobject.Slug(sd)
	if err != nil {
		return err
	}

	slug, err = c.Repo.CheckSlugForDuplicate("SubDomain", slug)
	if err != nil {
		return err
	}

	sd.SetSlug(slug)

	return c.UpdateContentObject(sd)
}

func (c *Content) CreateSubDomain(sd *valueobject.SubDomain) (string, error) {
	return c.newContent("SubDomain", sd)
}
