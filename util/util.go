package util

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
)

func OpenDB(path string, buckets []string, readOnly bool) *bolt.DB {
	db, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {
		log.Fatal(err)
	}

	if len(buckets) > 0 {
		CreateBucketsDB(db, buckets)
	}

	return db
}

func CreateBucketsDB(db *bolt.DB, buckets []string) {
	db.Update(func(tx *bolt.Tx) error {
		for _, name := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return fmt.Errorf("create bucket: %s %s", name, err)
			}
		}
		return nil
	})
}

func NumDecPlaces(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}
