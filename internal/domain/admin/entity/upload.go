package entity

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/mdfriday/hugoverse/internal/domain/admin/repository"
	"github.com/mdfriday/hugoverse/internal/domain/admin/valueobject"
	"github.com/mdfriday/hugoverse/internal/domain/content/factory"
	"net/url"
)

type Upload struct {
	Repo repository.Repository
}

func (a *Upload) UploadCreator() func() interface{} {
	return func() interface{} { return new(valueobject.FileUpload) }
}

func (a *Upload) GetUpload(id string) ([]byte, error) {
	return a.Repo.GetUpload(id)
}

func (a *Upload) DeleteUpload(id string) error {
	return a.Repo.DeleteUpload(id)
}

func (a *Upload) AllUploads() ([][]byte, error) {
	return a.Repo.AllUploads()
}

func (a *Upload) NewUpload(data url.Values) (int, error) {
	var upload valueobject.FileUpload

	decoder := schema.NewDecoder()
	decoder.SetAliasTag("json")     // allows simpler struct tagging when creating a content type
	decoder.IgnoreUnknownKeys(true) // will skip over form values submitted, but not in struct
	if err := decoder.Decode(&upload, data); err != nil {
		return 0, err
	}

	item, err := factory.NewItem()
	if err != nil {
		return 0, err
	}
	upload.Item = *item

	slug, err := a.Repo.CheckSlugForDuplicate("__uploads", upload.Name)
	if err != nil {
		return 0, err
	}
	upload.Slug = slug

	nextId, err := a.Repo.NextUploadId()
	if err != nil {
		return 0, err
	}
	upload.ID = int(nextId)

	uploadData, err := json.Marshal(upload)
	if err != nil {
		return 0, err
	}

	return upload.ID, a.Repo.NewUpload(fmt.Sprintf("%d", upload.ID), slug, uploadData)
}
