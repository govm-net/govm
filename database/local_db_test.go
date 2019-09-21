package database

import (
	"bytes"
	"log"
	"os"
	"path"
	"testing"
)

func TestNewLDB(t *testing.T) {
	fn := path.Join(gDbRoot, "aaa.db")
	os.Remove(fn)
	db := NewLDB("aaa.db")
	defer db.Close()
	tbn := []byte("tbname")
	key := []byte("key")
	value := []byte("value")
	v := db.LGet(1, tbn, key)
	if v != nil {
		t.Error("hope nil,but get value")
	}
	db.LSet(1, tbn, key, value)
	v = db.LGet(1, tbn, key)
	if bytes.Compare(value, v) != 0 {
		log.Printf("hope:%x,get:%x\n", value, v)
		t.Error("error")
	}
}
