package hatchery

import (
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
)

// BoltDBHeap is a Heap implementation backed by BoltDB.
type BoltDBHeap struct {
	// Path is the file path that the BoltDB file will live.
	// If a DB doesn't already exist at this path, it will be
	// created automatically. Otherwise, it will just be used
	// as-is.
	Path string

	once sync.Once
	db   *bolt.DB
}

// Put stores the kvp in the given BoltDB bucket. If the bucket doesn't
// already exist, it will be created automatically. If the key already exists
// in the bucket, it will be overwritten. An error is returned if the bucket
// could not be created, or the insertaion fails for whatever reason.
func (c *BoltDBHeap) Put(bucket, key string, value []byte) error {
	if err := c.initOnce(); err != nil {
		return err
	}
	err := c.db.Update(func(tx *bolt.Tx) error {
		buck, e := tx.CreateBucketIfNotExists([]byte(bucket))
		if e != nil {
			return e
		}
		return buck.Put([]byte(key), value)
	})
	if err != nil {
		return fmt.Errorf("put failed: %s", err)
	}
	return nil
}

// Get returns the value for the provided key and bucket. If the bucket doesn't
// already exist, it will be created automatically. ErrHeapNotExist is returned if
// No entry in the heap bucket for the requested key. Otherwise, an error is returned
// only if the bucket could not be created.
func (c *BoltDBHeap) Get(bucket, key string) ([]byte, error) {
	if err := c.initOnce(); err != nil {
		return nil, err
	}
	var b []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		buck, e := tx.CreateBucketIfNotExists([]byte(bucket))
		if e != nil {
			return e
		}
		vb := buck.Get([]byte(key))
		if vb == nil {
			return ErrHeapNotExist
		}
		b = make([]byte, len(vb))
		copy(b, vb)
		return nil
	})
	return b, err
}

// GetAll returns all heap entries in the given bucket. If the bucket doesn't
// already exist, it will be created automatically. An error is only returned if
// the bucket cannot be created.
func (c *BoltDBHeap) GetAll(bucket string) (map[string][]byte, error) {
	if err := c.initOnce(); err != nil {
		return nil, err
	}
	heap := make(map[string][]byte)
	err := c.db.View(func(tx *bolt.Tx) error {
		buck, e := tx.CreateBucketIfNotExists([]byte(bucket))
		if e != nil {
			return e
		}

		curr := buck.Cursor()
		for {
			k, v := curr.Next()
			if k == nil || v == nil {
				break
			}
			kc := make([]byte, len(k))
			copy(kc, k)
			vc := make([]byte, len(v))
			copy(vc, v)
			heap[string(kc)] = vc
		}
		return nil
	})
	return heap, err
}

// Close closes the BoltDB handle.
func (c *BoltDBHeap) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *BoltDBHeap) initOnce() error {
	var err error
	c.once.Do(func() {
		c.db, err = bolt.Open(c.Path, 0600, nil)
		if err != nil {
			return
		}
	})
	if err != nil {
		return fmt.Errorf("failed to open db at path %s: %s", c.Path, err)
	}
	return nil
}
