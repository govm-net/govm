package database

import (
	"github.com/boltdb/bolt"
	"log"
	"os"
	"path"
	"sync"
)

// LDB Local DB
type LDB struct {
	wmu sync.Mutex
	ldb *bolt.DB
}

// NewLDB new local db
func NewLDB(name string) *LDB {
	os.Mkdir(gDbRoot, 0600)
	out := LDB{}
	var err error
	fn := path.Join(gDbRoot, name)
	out.ldb, err = bolt.Open(fn, 0600, nil)
	if err != nil {
		log.Println("fail to open file(ldb):", fn, err)
		return nil
	}
	return &out
}

// Close close ldb
func (d *LDB) Close() {
	d.ldb.Close()
}

// LGet Local DB Get
func (d *LDB) LGet(chain uint64, tbName, key []byte) []byte {
	var out []byte
	tn := itoa(chain)
	tn = append(tn, 'a')
	tn = append(tn, tbName...)
	d.ldb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(tn)
		if b == nil {
			return nil
		}
		v := b.Get(key)
		if v != nil {
			out = make([]byte, len(v))
			copy(out, v)
		}
		return nil
	})
	return out
}

// LSet Local DB Set,if value = nil,will delete the key
func (d *LDB) LSet(chain uint64, tbName, key, value []byte) error {
	tn := itoa(chain)
	tn = append(tn, 'a')
	tn = append(tn, tbName...)
	d.wmu.Lock()
	defer d.wmu.Unlock()
	d.ldb.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tn)
		if err != nil {
			log.Printf("fail to create bucket,%x\n", tbName)
			return err
		}
		if value == nil {
			b.Delete(key)
		} else {
			err = b.Put(key, value)
			if err != nil {
				log.Println("fail to put:", err)
				return err
			}
		}
		return nil
	})
	return nil
}
