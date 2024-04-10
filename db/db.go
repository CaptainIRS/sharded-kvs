package db

import (
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var defaultBucket = []byte("default")

/*
 * Separate package - so that the DB code doesn't interfere with the HTTPHandler code
 * and vice-versa...
 */

type Database struct {
	db *bolt.DB
}

// returns an instance of the DB that we want to work with!
func (d *Database) createDefaultBucket() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
	})
}

func NewDatabase(dbPath string) (db *Database, closeFunc func() error, err error) {
	boltDB, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, nil, err
	}
	db = &Database{db: boltDB}
	closeFunc = boltDB.Close
	if err := db.createDefaultBucket(); err != nil {
		return nil, nil, fmt.Errorf("creating default bucket: %w", err)
	}
	return db, closeFunc, nil
}

// sets key to the provided value into the default DB or returns an error
func (d *Database) SetKey(key string, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		return b.Put([]byte(key), value)
	})
}

// gets the value of the requested key from the default DB
func (d *Database) GetKey(key string) ([]byte, error) {
	var result []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(defaultBucket)
		result = b.Get([]byte(key))
		return nil
	})
	if err == nil {
		return result, nil
	}
	return nil, err
}
