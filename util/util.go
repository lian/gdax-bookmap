package util

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
)

func OpenDB(path string, buckets []string, readOnly bool) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {

		path, err = osxBundlePath(path)
		if err != nil {
			return nil, err
		}

		db, err = bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly})
		if err != nil {
			return nil, err
		}
	}

	if len(buckets) > 0 {
		CreateBucketsDB(db, buckets)
	}

	return db, nil
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

func osxBundlePath(db_path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	path := filepath.Join(usr.HomeDir, "Library/Application Support")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", err
	}

	path = filepath.Join(path, "gdax-bookmap")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0755); err != nil {
			return "", err
		}
	}

	return filepath.Join(path, db_path), nil
}

func NumDecPlaces(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}
