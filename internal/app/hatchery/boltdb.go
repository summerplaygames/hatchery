package hatchery

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/boltdb/bolt"
)

type BoltDBHeap struct {
	Path string
	once sync.Once
	db   *bolt.DB
}

func (c *BoltDBHeap) Put(bucket string, key, value interface{}) error {
	if err := c.initOnce(); err != nil {
		return err
	}
	vb, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value JSON: %s", err)
	}
	kb, err := c.getKeyBytes(key)
	if err != nil {
		return fmt.Errorf("failed to marshal key to bytes: %s", err)
	}
	err = c.db.Update(func(tx *bolt.Tx) error {
		buck, e := tx.CreateBucketIfNotExists([]byte(bucket))
		if e != nil {
			return e
		}
		return buck.Put(kb, vb)
	})
	if err != nil {
		return fmt.Errorf("put failed: %s", err)
	}
	return nil
}

func (c *BoltDBHeap) Get(bucket string, key interface{}, typ reflect.Type) (interface{}, error) {
	if err := c.initOnce(); err != nil {
		return nil, err
	}
	kb, err := c.getKeyBytes(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key to bytes: %s", err)
	}
	var i interface{}
	err = c.db.View(func(tx *bolt.Tx) error {
		buck, e := tx.CreateBucketIfNotExists([]byte(bucket))
		if e != nil {
			return e
		}
		vb := buck.Get(kb)
		ptr := reflect.New(typ)
		i = ptr.Elem().Interface()
		return json.Unmarshal(vb, i)
	})
	return i, err
}

func (c *BoltDBHeap) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *BoltDBHeap) getKeyBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, key)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes(), nil
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
