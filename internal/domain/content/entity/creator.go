package entity

import (
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/gorilla/schema"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/form"
	"net/url"
	"strconv"
)

func (c *Content) NewContent(contentType string, data url.Values) (string, error) {
	t, ok := c.GetContentCreator(contentType)
	if !ok {
		return "", errors.New("invalid content type")
	}
	ci := t()

	d, err := form.Convert(data)
	if err != nil {
		return "", err
	}
	// Decode Content
	dec := schema.NewDecoder()
	dec.SetAliasTag("json")     // allows simpler struct tagging when creating a content type
	dec.IgnoreUnknownKeys(true) // will skip over form values submitted, but not in struct
	err = dec.Decode(ci, d)
	if err != nil {
		return "", err
	}

	return c.newContent(contentType, ci)
}

func (c *Content) newContent(contentType string, ci any) (string, error) {
	cii, ok := ci.(content.Identifiable)
	if ok {
		uid, err := uuid.NewV4()
		if err != nil {
			return "", err
		}
		cii.SetUniqueID(uid)

		id, err := c.Repo.NextContentId(contentType)
		if err != nil {
			return "", err
		}
		cii.SetItemID(int(id))
	} else {
		return "", errors.New("content type does not implement Identifiable")
	}

	slug, err := valueobject.Slug(cii)
	if err != nil {
		return "", err
	}

	slug, err = c.Repo.CheckSlugForDuplicate(contentType, slug)
	if err != nil {
		return "", err
	}

	ciSlug, ok := ci.(content.Sluggable)
	if ok {
		ciSlug.SetSlug(slug)
	} else {
		return "", errors.New("content type does not implement Sluggable")
	}

	cis, ok := ci.(content.Statusable)
	if ok {
		if cis.ItemStatus() == "" {
			cis.SetItemStatus(content.Public)
		}
	} else {
		return "", errors.New("content type does not implement Statusable")
	}

	cih, ok := ci.(content.Hashable)
	if ok {
		cih.SetHash()
	}

	meta, ok := ci.(content.Metable)
	if ok {
		if err := meta.SetMeta(c.Hugo.DirService); err != nil {
			c.Log.Errorf("failed to set %s meta: %v", contentType, err)
		}
	}

	b, err := c.Marshal(ci)
	if err != nil {
		return "", err
	}

	if err := c.Repo.NewContent(ci, b); err != nil {
		return "", err
	}

	if cis.ItemStatus() == content.Public {
		go func() {
			if err := c.SortContent(contentType); err != nil {
				c.Log.Errorln("sort content err: ", err)
			}
		}()
	}

	id := int64(cii.ItemID())

	go func() {
		// update data in search index
		if err := c.Search.UpdateIndex(
			GetNamespace(contentType, string(cis.ItemStatus())),
			fmt.Sprintf("%d", id), b); err != nil {

			c.Log.Println("[search] UpdateIndex Error:", err)
		}
	}()

	return strconv.FormatInt(id, 10), nil
}

func (c *Content) syncCheck(sp *valueobject.SitePost) {
	siteId, err := valueobject.GetIdFromQueryString(sp.Site)
	if err != nil {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	postId, err := valueobject.GetIdFromQueryString(sp.Post)
	if err != nil {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	s, err := c.getContent("Site", siteId)
	if err != nil {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	p, err := c.getContent("Post", postId)
	if err != nil {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	site, ok := s.(*valueobject.Site)
	if !ok {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	post, ok := p.(*valueobject.Post)
	if !ok {
		c.Log.Println("syncCheck error: ", err)
		return
	}

	if err := c.Hugo.syncPostToFilesystem(site, post, sp); err != nil {
		c.Log.Println("syncCheck error: ", err)
		return
	}
}
