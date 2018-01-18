package websocket

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

func OpenDB(path string, buckets []string, readOnly bool) *bolt.DB {
	db, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		for _, name := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return fmt.Errorf("create bucket: %s %s", name, err)
			}
		}
		return nil
	})

	return db
}
