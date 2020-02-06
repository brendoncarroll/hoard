package boltkv

import bolt "go.etcd.io/bbolt"

type BoltKV struct {
	db     *bolt.DB
	bucket []byte
}

func New(db *bolt.DB, bucketName string) BoltKV {
	return BoltKV{
		db:     db,
		bucket: []byte(bucketName),
	}
}

func (b BoltKV) Get(key []byte) ([]byte, error) {
	var data []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(b.bucket)
		if b == nil {
			return nil
		}
		value := b.Get(key)
		data = append([]byte{}, value...)
		return nil
	})

	return data, err
}

func (b BoltKV) Put(key, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(b.bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}
