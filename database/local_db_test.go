package database

import (
	"bytes"
	"log"
	"os"
	"path"
	"testing"
)

func TestLDB1(t *testing.T) {
	fn := path.Join(gDbRoot, "aaa.db")
	os.Remove(fn)
	db := NewLDB("aaa.db", 0)
	defer db.Close()
	tbn := "tbname"
	key := []byte("key")
	value := []byte("value")
	v := db.LGet(1, tbn, key)
	if v != nil {
		t.Error("hope nil,but get value")
	}
	db.LSet(1, tbn, key, value)
	if db.wdisk != 1 {
		t.Error("hope write disk one time")
	}
	v = db.LGet(1, tbn, key)
	if bytes.Compare(value, v) != 0 {
		log.Printf("hope:%x,get:%x\n", value, v)
		t.Error("error")
	}
}

func TestLDB2(t *testing.T) {
	fn := path.Join(gDbRoot, "aaa.db")
	os.Remove(fn)
	db := NewLDB("aaa.db", 100)
	defer db.Close()
	tbn := "tbname"
	key := []byte("key")
	value := []byte("value")
	db.SetCache(tbn)
	v := db.LGet(1, tbn, key)
	if v != nil {
		t.Error("hope nil,but get value")
	}
	if db.rdisk != 1 {
		t.Error("hope read disk one time")
	}
	v = db.LGet(1, tbn, key)
	if v != nil || db.rdisk != 1 {
		t.Error("hope read disk one time,cache")
	}
	db.LSet(1, tbn, key, value)
	if db.wdisk != 1 {
		t.Error("hope write disk one time")
	}
	db.LSet(1, tbn, key, value)
	if db.wdisk != 1 {
		t.Error("hope write disk one time,same value")
	}

	v = db.LGet(1, tbn, key)
	if bytes.Compare(value, v) != 0 {
		log.Printf("hope:%x,get:%x\n", value, v)
		t.Error("error")
	}
	if db.rdisk != 1 {
		t.Error("hope read from cache")
	}
}
