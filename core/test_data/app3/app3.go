package app

import (
	core "github.com/govm-net/govm/core/test_data/zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
)

type tApp struct {
	Ops   uint8
	Key   [4]byte
	Value uint64
	Other uint64
}

// ops
const (
	OpsWriteData = uint8(iota)
	OpsReadData
	OpsWriteLog
	OpsReadLog
)

func run(user, in []byte, cost uint64) {
	info := tApp{}
	core.Event(info, "start app3", user, in)
	defer core.Event(info, "finish app3", user, in)
	core.Decode(0, in, &info)
	switch info.Ops {
	case OpsWriteData:
		db := core.GetDB(info)
		db.SetInt(info.Key[:], info.Value, 30*24*3600*1000)
		v := db.GetInt(info.Key[:])
		if v != info.Value {
			panic("fail to write data")
		}
	case OpsReadData:
		db := core.GetDB(info)
		v := db.GetInt(info.Key[:])
		if v != info.Value {
			panic("different value")
		}
	case OpsWriteLog:
		log := core.GetLog(info)
		log.Write(info.Key[:], core.Encode(0, info.Value))
		data := log.Read(0, info.Key[:])
		if data == nil {
			panic("error data")
		}
	case OpsReadLog:
		log := core.GetLog(info)
		data := log.Read(info.Other, info.Key[:])
		if len(data) == 0 && info.Value == 0 {
			return
		}
		if len(data) != 8 {
			panic("error data length")
		}
		var v uint64
		core.Decode(0, data, &v)
		if v != info.Value {
			panic("different value")
		}
	}
}
