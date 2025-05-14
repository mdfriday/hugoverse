package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/internal/domain/content/repository"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/pkg/form"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"log"
	"net/url"
	"sort"
)

type contentSvc interface {
	newContent(contentType string, ci any) (string, error)
	search(contentType string, query string) ([][]byte, error)
}

type Content struct {
	UserTypes  map[string]content.Creator
	AdminTypes map[string]content.Creator
	Repo       repository.Repository

	*Search
	*Hugo

	Log loggers.Logger
}

func (c *Content) AllContents(contentType string) [][]byte {
	return c.Repo.AllContent(contentType)
}

func (c *Content) GetContents(ids []content.Identifier) ([][]byte, error) {
	var contents [][]byte
	for _, id := range ids {
		data, err := c.GetContent(id.ContentType(), id.ID(), "")
		if err != nil {
			return nil, err
		}
		if data == nil {
			c.Log.Infof("Content not found: %s %s", id.ContentType(), id.ID())
			continue
		}

		contents = append(contents, data)
	}
	return contents, nil
}

func (c *Content) GetContent(contentType, id, status string) ([]byte, error) {
	return c.Repo.GetContent(GetNamespace(contentType, status), id)
}

func (c *Content) GetRandomContent(contentType, status string) ([]byte, error) {
	return c.Repo.GetRandomContent(GetNamespace(contentType, status))
}

func (c *Content) GetContentByHash(contentType, hash, status string) ([]byte, error) {
	idb, err := c.Repo.GetIdByHash(GetNamespace(contentType, status), hash)
	if err != nil {
		return nil, err
	}
	id := string(idb)
	return c.GetContent(contentType, id, status)
}

func (c *Content) DeleteContent(contentType, id, status string) error {
	data, err := c.GetContent(contentType, id, status)
	if err != nil {
		return err
	}
	ct, ok := c.GetContentCreator(contentType)
	if !ok {
		return errors.New("invalid content type")
	}
	cti := ct()
	if err = json.Unmarshal(data, cti); err != nil {
		return err
	}

	ns := GetNamespace(contentType, status)

	hash := ""
	if ctiHash, ok := cti.(content.Hashable); ok {
		hash = ctiHash.ItemHash()
	}
	if err := c.Repo.DeleteContent(ns, id, cti.(content.Sluggable).ItemSlug(), hash); err != nil {
		return err
	}

	go func() {
		// delete indexed data from search index
		if isPublicNamespace(ns) {
			err = c.Search.DeleteIndex(ns)
			if err != nil {
				log.Println("[search] DeleteIndex Error:", err)
			}
		}
	}()

	if err := c.SortContent(contentType); err != nil {
		return err
	}

	return nil
}

func (c *Content) DeleteContentObject(ci any) error {
	cis, ok := ci.(content.Statusable)
	if !ok {
		return errors.New("invalid content type")
	}

	cii, ok := ci.(content.Identifiable)
	ns := GetNamespace(cii.ItemName(), string(cis.ItemStatus()))

	hash := ""
	if ctiHash, ok := ci.(content.Hashable); ok {
		hash = ctiHash.ItemHash()
	}
	if err := c.Repo.DeleteContent(ns, fmt.Sprintf("%d", cii.ItemID()), ci.(content.Sluggable).ItemSlug(), hash); err != nil {
		return err
	}

	go func() {
		// delete indexed data from search index
		if isPublicNamespace(ns) {
			err := c.Search.DeleteIndex(ns)
			if err != nil {
				log.Println("[search] DeleteIndex Error:", err)
			}
		}
	}()

	if err := c.SortContent(cii.ItemName()); err != nil {
		return err
	}

	return nil
}

func (c *Content) UpdateContent(contentType string, data url.Values) error {
	t, ok := c.GetContentCreator(contentType)
	if !ok {
		return errors.New("invalid content type")
	}
	ci := t()

	d, err := form.Convert(data)
	if err != nil {
		return err
	}
	// Decode Content
	dec := schema.NewDecoder()
	dec.SetAliasTag("json")     // allows simpler struct tagging when creating a content type
	dec.IgnoreUnknownKeys(true) // will skip over form values submitted, but not in struct
	err = dec.Decode(ci, d)
	if err != nil {
		return err
	}

	return c.UpdateContentObject(ci)
}

func (c *Content) UpdateContentObject(ci any) error {
	b, err := c.Marshal(ci)
	if err != nil {
		return err
	}

	if err := c.Repo.PutContent(ci, b); err != nil {
		c.Log.Errorln("[repo] PutContent Error:", err)
		return err
	}

	cis, ok := ci.(content.Statusable)
	if !ok {
		return errors.New("invalid content type")
	}
	status := cis.ItemStatus()

	cih, ok := ci.(content.Hashable)
	if ok {
		cih.SetHash()
	}

	cii, ok := ci.(content.Identifiable)
	if ok {
		go func() {
			// update data in search index
			if err := c.Search.UpdateIndex(
				GetNamespace(cii.ItemName(), string(cis.ItemStatus())),
				fmt.Sprintf("%d", cii.ItemID()), b); err != nil {

				c.Log.Errorln("[search] UpdateIndex Error:", err)
			}
		}()
	}

	if status == content.Public {
		go func() {
			err := c.SortContent(cii.ItemName())
			if err != nil {
				c.Log.Errorln("sort content err: ", err)
			}
		}()
	}

	return nil
}

func (c *Content) SortContent(contentType string) error {
	// wait if running too frequently per namespace
	if !valueobject.EnoughTime(contentType, c.SortContent) {
		return nil
	}

	t, ok := c.GetContentCreator(contentType)
	if !ok {
		return errors.New("invalid content type")
	}

	all := c.Repo.AllContent(contentType)

	var posts valueobject.SortableContent
	var err error
	// decode each (json) into type to then sort
	for i := range all {
		j := all[i]
		if j == nil {
			c.Log.Errorln("Error decoding json while sorting", contentType, ": nil")
			continue
		}

		post := t()
		unmarshal, ok := post.(content.Unmarshalable)
		if ok {
			err = unmarshal.UnmarshalJSON(j)
		} else {
			err = json.Unmarshal(j, &post)
		}
		if err != nil {
			c.Log.Errorln("Error decoding json while sorting", contentType, ":", err, string(j))
			continue
		}

		posts = append(posts, post.(content.Sortable))
	}

	// sort posts
	sort.Sort(posts)

	// marshal posts to json
	var bb [][]byte
	for i := range posts {
		j, err := json.Marshal(posts[i])
		if err != nil {
			// log error and kill sort so __sorted is not in invalid state
			log.Println("Error marshal post to json in SortContent:", err)
			return err
		}

		bb = append(bb, j)
	}

	m := make(map[string][]byte)
	// encode to json and store as 'post.Time():i':post
	for i := range bb {
		cid := fmt.Sprintf("%d:%d", posts[i].Time(), i)
		m[cid] = bb[i]
	}

	// store in <namespace>_sorted bucket, first delete existing
	if err := c.Repo.PutSortedContent(contentType, m); err != nil {
		return err
	}

	return nil
}

func (c *Content) Marshal(content any) ([]byte, error) {
	j, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (c *Content) Unmarshal(data []byte, content any) error {
	return json.Unmarshal(data, content)
}

func (c *Content) NormalizeString(s string) (string, error) {
	return valueobject.StringToSlug(s)
}
