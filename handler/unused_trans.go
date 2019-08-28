package handler

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	core "github.com/lengzhao/govm/core"
	"log"
)

type transItem struct {
	Time   uint64
	Energy uint64
	Size   uint64
	User   []byte
	Key    []byte
}

const (
	limitSize = 900 * 1024
)

var dataDb *bolt.DB

func init() {
	var err error
	dataDb, err = bolt.Open("unused_trans.db", 0600, nil)
	if err != nil {
		log.Println("fail to open db file:", "unused_trans.db", err)
		return
	}
}

func getTbName(chain uint64) []byte {
	str := fmt.Sprintf("tbTransInfo_%d", chain)
	return []byte(str)
}

func addTrans(key, user []byte, chain, Time, Energy, Size uint64) error {
	item := transItem{Time, Energy, Size, user, key}
	data, _ := json.Marshal(item)
	return dataDb.Update(func(tx *bolt.Tx) error {
		buck, err := tx.CreateBucketIfNotExists(getTbName(chain))
		if err != nil {
			return err
		}
		return buck.Put(key, data)
	})
}

func delTrans(chain uint64, key []byte) {
	dataDb.Update(func(tx *bolt.Tx) error {
		buck := tx.Bucket(getTbName(chain))
		if buck == nil {
			return nil
		}
		buck.Delete(key)
		return nil
	})
}

func itoa(in uint64) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, in)
	return out
}

func atoi(in []byte) uint64 {
	return binary.BigEndian.Uint64(in)
}

func getTransListForMine(chain uint64) [][]byte {
	out := make([][]byte, 0)
	delList := make([][]byte, 0)
	var size uint64
	maxT := core.GetBlockTime(chain)
	minT := maxT - 20*24*3600*1000
	// acceptT := core.GetBlockTime(chain) - 20*24*3600*1000
	dataDb.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket(getTbName(chain))
		if buck == nil {
			return nil
		}
		c := buck.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			item := new(transItem)
			json.Unmarshal(v, item)
			if item.Time < minT {
				delList = append(delList, k)
				log.Printf("transaction too old,not accept. key:%x,time:%d, accept time:%d\n", item.Key, item.Time, minT)
				continue
			}
			if item.Time > maxT{
				continue
			}
			if size+item.Size > limitSize {
				return nil
			}
			out = append(out, k)
			size += item.Size
		}
		return nil
	})

	dataDb.Update(func(tx *bolt.Tx) error {
		buck := tx.Bucket(getTbName(chain))
		for _, k := range delList {
			buck.Delete(k)
		}
		for _, k := range out {
			buck.Delete(k)
		}
		return nil
	})
	log.Printf("mine. trans list len:%d\n", len(out))

	return out
}
