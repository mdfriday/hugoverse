package db

import (
	"bytes"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"math/rand"
)

func All(item Item) ([][]byte, error) {
	var items [][]byte
	err := store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(item.Bucket()))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		numKeys := b.Stats().KeyN
		items = make([][]byte, 0, numKeys)

		return b.ForEach(func(k, v []byte) error {
			items = append(items, v)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) Get(item Item) ([]byte, error) {
	val := &bytes.Buffer{}
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(item.Bucket()))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		obj := b.Get([]byte(item.Key()))

		_, err := val.Write(obj)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if val.Bytes() == nil {
		return nil, nil
	}

	return val.Bytes(), nil
}

func (s *Store) GetRandom(item Item) ([]byte, error) {
	val := &bytes.Buffer{}
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(item.Bucket()))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		// 使用 Cursor 遍历所有 key
		var keys [][]byte
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, k)
		}

		if len(keys) == 0 {
			return fmt.Errorf("no images found in bucket %s", item.Bucket())
		}

		// 直接用 rand.Intn 选一个 key（不需要 Seed）
		randomKey := keys[rand.Intn(len(keys))]
		obj := b.Get(randomKey)

		_, err := val.Write(obj)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if val.Bytes() == nil {
		return nil, nil
	}

	return val.Bytes(), nil
}

func (s *Store) Set(item Item) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(item.Bucket()))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(item.Key()), item.Value())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) Delete(item Item) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(item.Bucket()))
		if b == nil {
			return bolt.ErrBucketNotFound
		}

		err := b.Delete([]byte(item.Key()))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
