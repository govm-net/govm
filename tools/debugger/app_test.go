package a1000000000000000000000000000000000000000000000000000000000000000

import (
	"encoding/hex"
	"fmt"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/runtime"
	"testing"
	"time"
)

var blockKey = [32]byte{1, 2, 3, 4, 5}
var chain uint64 = 1

func TestMain(m *testing.M) {
	m.Run()
	// database.Commit(chain, key)
}

func hexToBytes(in string) []byte {
	out, err := hex.DecodeString(in)
	if err != nil {
		fmt.Println("fail to decode hex:", err)
		panic(err)
	}
	return out
}

func intToBytes(in uint64) []byte {
	return runtime.Encode(in)
}

func appInfo() []byte {
	type AppInfo struct {
		Account [24]byte
		LineSum uint64
		Life    uint64
		Flag    uint8
	}
	info := AppInfo{}
	runtime.Decode(hexToBytes("ff0000000000000000000000000000000000000000000000"), &info.Account)
	info.Life = uint64(time.Now().Add(time.Hour*24*365*2).Unix()) * 1000
	info.Flag = 15
	return runtime.Encode(info)
}

func blockInfo() []byte {
	// TRunParam Run接口的入参
	type BaseInfo struct {
		Key           [32]byte
		Time          uint64
		Chain         uint64
		ID            uint64
		BaseOpsEnergy uint64
		Producer      [24]byte
		ParentID      uint64
		LeftChildID   uint64
		RightChildID  uint64
	}
	param := BaseInfo{blockKey,
		uint64(time.Now().Unix() * 1000), 1, 100, 10,
		[24]byte{1, 2}, 0, 0, 0}
	return runtime.Encode(param)
}

type SubData struct {
	Describe   string
	Chain      uint64
	AppName    string
	StructName string
	Key        []byte
	Value      []byte
	Life       uint64 //Hour
}

// TestImitateData imitate data to database
func TestImitateData(t *testing.T) {
	datas := []SubData{
		{"2.set cost of app account",
			chain, "ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c",
			"dbCoin", hexToBytes("ff0000000000000000000000000000000000000000000000"),
			intToBytes(1000000000000000), 100},
		{"3.set app info",
			chain, "ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c",
			"dbApp", hexToBytes("1000000000000000000000000000000000000000000000000000000000000000"),
			appInfo(), 100},
		{"4.set block info",
			chain, "ae4a05b2b8a4de21d9e6f26e9d7992f7f33e89689f3015f3fc8a3a3278815e28c",
			"dbStat", []byte{0}, blockInfo(), 100},
	}
	now := uint64(time.Now().Unix()) * 1000
	for _, d := range datas {
		if len(d.AppName) != 65 {
			t.Fatal("error app name,len != 64:", d.AppName)
		}
		tbName := []byte(d.AppName + "." + d.StructName)
		life := now + d.Life*3600*1000
		value := append(d.Value, runtime.Encode(life)...)
		tbName[0] = 'd'
		err := database.GetClient().Set(d.Chain, tbName, d.Key, value)
		if err != nil {
			fmt.Println("fail to imitate data.", d.Describe)
			t.Fatal(err)
		}
	}
}

func Test_run(t *testing.T) {
	fmt.Println("testing start", t.Name())
	c := conf.GetConf()
	if c.DbServerAddr != "127.0.0.1:12345" {
		// 为避免不小心测试是操作到正式的db server，增加了该限制。
		t.Fatal("wrong db server address,error test dir?")
	}
	key := blockKey[:]
	client := database.GetClient()
	err := client.OpenFlag(chain, key)
	if err != nil {
		// fmt.Println("fail to open Flag,", err)
		f := client.GetLastFlag(chain)
		client.Cancel(chain, f)
		client.OpenFlag(chain, key)
	}
	defer client.Cancel(chain, key)

	run(hexToBytes("02984010319cd34659f7fcb20b31d615d850ab32ca930618"), []byte("parament"), 10)
}
