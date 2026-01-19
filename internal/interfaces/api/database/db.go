package database

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content"
	"github.com/mdfriday/hugoverse/pkg/db"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"path"
	"strconv"
	"strings"
)

var (
	adminOriginBuckets = []string{
		"__config", "__users",
		"__contentIndex",
	}

	userBuckets = []string{
		"__addons", "__uploads",
		"__contentIndex",
	}
)

type Database struct {
	dataDir string
	userDir string

	contentBuckets []string
	adminBuckets   []string

	adminStore *db.Store
	userStore  *db.Store

	log loggers.Logger
}

func New(dataDir string) (*Database, error) {
	return &Database{
		dataDir: dataDir,
		log:     loggers.NewDefault(),
	}, nil
}

func (d *Database) UserDataDir() string {
	return path.Join(d.dataDir, d.userDir)
}

func (d *Database) AdminDataDir() string {
	return d.dataDir
}

func (d *Database) UserDir() string {
	return d.userDir
}

func (d *Database) RegisterContentBuckets(contentTypeNames []string) {
	d.contentBuckets = append(d.contentBuckets, contentTypeNames...)
}

func (d *Database) Close() {
	if d.userStore != nil {
		d.userStore.Close()
	}
	if d.adminStore != nil {
		d.adminStore.Close()
	}
}

func (d *Database) getStore(ns string) *db.Store {
	for _, bucket := range d.adminBuckets {
		if strings.HasPrefix(ns, bucket) {
			return d.adminStore
		}
	}

	return d.userStore
}

func (d *Database) PutSortedContent(namespace string, m map[string][]byte) error {
	return d.getStore(namespace).Sort(newItems(namespace, m))
}

func (d *Database) AllContent(namespace string) [][]byte {
	return d.getStore(namespace).ContentAll(namespace)
}

func (d *Database) ContentByPrefix(namespace, prefix string) ([][]byte, error) {
	return d.getStore(namespace).ContentByPrefix(bucketNameWithIndex(namespace), prefix)
}

func (d *Database) GetContent(namespace string, id string) ([]byte, error) {
	return d.getStore(namespace).Get(
		&item{
			bucket: namespace,
			key:    id,
		})
}

func (d *Database) GetRandomContent(namespace string) ([]byte, error) {
	return d.getStore(namespace).GetRandom(
		&item{
			bucket: namespace,
		})
}

func (d *Database) DeleteContent(namespace string, id string, slug string, hash string) error {
	if err := d.getStore(namespace).Delete(&item{bucket: namespace, key: id}); err != nil {
		return err
	}

	if err := d.getStore(namespace).Delete(&item{bucket: bucketNameWithIndex(namespace), key: slug}); err != nil {
		return err
	}

	if err := d.getStore(namespace).RemoveIndex(slug); err != nil {
		return err
	}

	if hash != "" {
		if err := d.getStore(namespace).RemoveIndex(fmt.Sprintf("%s:%s", namespace, hash)); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) PutContent(ci any, data []byte) error {
	cii, ok := ci.(content.Identifiable)
	if !ok {
		return errors.New("invalid content type")
	}
	id := cii.ItemID()
	ns := cii.ItemName()

	cis, ok := ci.(content.Statusable)
	if !ok {
		return errors.New("invalid content type")
	}
	status := cis.ItemStatus()

	bucket := ns
	if !(status == content.Public || status == "") {
		bucket = fmt.Sprintf("%s%s", ns, bucketNameWithPrefix(string(status)))
	}

	if err := d.getStore(ns).Set(
		&item{
			bucket: bucket,
			key:    strconv.FormatInt(int64(id), 10),
			value:  data,
		}); err != nil {
		return err
	}

	ciSlug, ok := ci.(content.Sluggable)
	if ok {
		if err := d.getStore(ns).Set(
			&item{
				bucket: bucketNameWithIndex(ns),
				key:    ciSlug.ItemSlug(),
				value:  data,
			}); err != nil {
			return err
		}

		if err := d.getStore(ns).SetIndex(newKeyValueItem(ciSlug.ItemSlug(), fmt.Sprintf("%s:%d", ns, id))); err != nil {
			return err
		}
	}

	ciHash, ok := ci.(content.Hashable)
	if ok {
		if err := d.getStore(ns).SetIndex(newKeyValueItem(fmt.Sprintf("%s:%s", ns, ciHash.ItemHash()), fmt.Sprintf("%d", id))); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) NewContent(ci any, data []byte) error {
	return d.PutContent(ci, data)
}

func (d *Database) NextContentId(ns string) (uint64, error) {
	return d.getStore(ns).NextSequence(&item{bucket: ns})
}

func (d *Database) NextUploadId() (uint64, error) {
	return d.userStore.NextSequence(newBucketItem("uploads"))
}

func (d *Database) GetUpload(id string) ([]byte, error) {
	key, err := keyBit8Uint64(id)
	if err != nil {
		return nil, err
	}
	return d.userStore.Get(newUploadItem(key, nil))
}

func (d *Database) DeleteUpload(id string) error {
	key, err := keyBit8Uint64(id)
	if err != nil {
		return err
	}
	return d.userStore.Delete(newUploadItem(key, nil))
}

func (d *Database) AllUploads() ([][]byte, error) {
	return db.All(newUploadItem("", nil))
}

func (d *Database) NewUpload(id, slug string, data []byte) error {
	key, err := keyBit8Uint64(id)
	if err != nil {
		return err
	}
	if err := d.userStore.Set(newUploadItem(key, data)); err != nil {
		return err
	}

	if err := d.userStore.SetIndex(newKeyValueItem(
		slug,
		fmt.Sprintf("%s:%s", bucketNameWithPrefix("upload"), id))); err != nil {
		return err
	}

	return nil
}

func keyBit8Uint64(sid string) (string, error) {
	id, err := strconv.Atoi(sid)
	if err != nil {
		return "", err
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(id))
	return string(b), err
}

func (d *Database) CheckSlugForDuplicate(namespace string, slug string) (string, error) {
	return d.getStore(namespace).CheckSlugForDuplicate(slug)
}

func (d *Database) GetIdByHash(namespace string, hash string) ([]byte, error) {
	return d.getStore(namespace).GetIndex(fmt.Sprintf("%s:%s", namespace, hash))
}

func (d *Database) Query(namespace string, opts db.QueryOptions) (int, [][]byte) {
	return d.getStore(namespace).Query(namespace, opts)
}
