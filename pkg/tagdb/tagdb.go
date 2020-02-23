package tagdb

import (
	"context"
	"encoding/binary"
	"errors"
	"log"

	"github.com/brendoncarroll/hoard/pkg/taggers"
	bolt "go.etcd.io/bbolt"
)

const bucketTags = "tags"

/* TagDB structures tags wil the following buckets
tags/
	<tag_key>/
		f/
			<entity> -> <tag_value>
			<entity> -> <tag_value>
			...
		i/
			<tag_value> -> <entity>
			<tag_value> -> <entity>
			...
	<tag_key>
*/
type TagDB struct {
	db *bolt.DB
}

func NewDB(db *bolt.DB) *TagDB {
	return &TagDB{db: db}
}

func (tdb *TagDB) PutTag(ctx context.Context, entity uint64, key, value string) error {
	err := tdb.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketTags))
		if err != nil {
			return err
		}
		forward, inverted, err := bucketsForTag(tx, key)
		if err != nil {
			return err
		}

		entityBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(entityBytes, entity)

		if err := inverted.Put(invKey([]byte(value), entityBytes), nil); err != nil {
			return err
		}
		if err := forward.Put(entityBytes, []byte(value)); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (tdb *TagDB) DeleteTag(ctx context.Context, entity uint64, key string) error {
	err := tdb.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(key))
		if b == nil {
			return nil
		}
		forward, inverted, err := bucketsForTag(tx, key)
		if err != nil {
			return err
		}
		entityBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(entityBytes, entity)

		value := forward.Get(entityBytes)
		if err := inverted.Delete(invKey(value, entityBytes)); err != nil {
			return err
		}
		if err := forward.Delete(entityBytes); err != nil {
			return err
		}
		return nil
	})

	return err
}

func (tdb *TagDB) GetTag(ctx context.Context, mID uint64, key string) (string, error) {
	var value string
	err := tdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(key))
		if b == nil {
			return nil
		}
		forward, _, err := bucketsForTag(tx, key)
		if err != nil {
			return err
		}
		value = string(forward.Get(idToBytes(mID)))
		return nil
	})
	return value, err
}

func (tdb *TagDB) AllTagsFor(ctx context.Context, entity uint64) (taggers.TagSet, error) {
	tags := make(taggers.TagSet)
	err := tdb.db.View(func(tx *bolt.Tx) error {
		allTags := tx.Bucket([]byte(bucketTags))

		c := allTags.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil {
				log.Println("WARN: found non bucket at key: ", string(k))
				continue
			}
			forward, _, err := bucketsForTag(tx, string(k))
			if err != nil {
				return err
			}
			entityBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(entityBytes, entity)
			tagValue := forward.Get(entityBytes)
			tags[string(k)] = string(tagValue)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func bucketsForTag(tx *bolt.Tx, key string) (forward, inverted *bolt.Bucket, err error) {
	allTags := tx.Bucket([]byte(bucketTags))

	var getBucket = func(b *bolt.Bucket, name []byte) (*bolt.Bucket, error) {
		return b.CreateBucketIfNotExists(name)
	}
	if !tx.Writable() {
		getBucket = func(b *bolt.Bucket, name []byte) (*bolt.Bucket, error) {
			b2 := b.Bucket(name)
			var err error
			if b2 == nil {
				err = bolt.ErrBucketNotFound
			}
			return b2, err
		}
	}

	tagB, err := getBucket(allTags, []byte(key))
	if err != nil {
		return nil, nil, err
	}
	forward, err = getBucket(tagB, []byte("f"))
	if err != nil {
		return nil, nil, err
	}
	inverted, err = getBucket(tagB, []byte("i"))
	if err != nil {
		return nil, nil, err
	}
	return forward, inverted, nil
}

func bytesToID(buf []byte) uint64 {
	return binary.BigEndian.Uint64(buf)
}

func idToBytes(x uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], x)
	return buf[:]
}

func invKey(value []byte, entitBytes []byte) []byte {
	return append(value, entitBytes...)
}

func splitInvKey(key []byte) (id uint64, value []byte, err error) {
	if len(key) < 9 {
		return 0, nil, errors.New("invalid key for inverted bucket")
	}
	id = bytesToID(key[:8])
	value = key[8:]
	return id, key, nil
}
