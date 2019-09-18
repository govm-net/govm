package database

import (
	"log"
	"net/rpc"

	"github.com/lengzhao/govm/conf"
)

var dbServer string
var addrType string
var client *rpc.Client
var lock chan int

func init() {
	c := conf.GetConf()
	dbServer = c.DbServerAddr
	addrType = c.DbAddrType
	lock = make(chan int, 1)
}

// OpenFlag 开启标志，标志用于记录操作，支持批量操作的回滚
func OpenFlag(chain uint64, flag []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := FlagArgs{chain, flag}
	var reply bool
	err = client.Call("TDb.OpenFlag", &args, &reply)
	if err != nil {
		log.Println("fail to OpenFlag:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// GetLastFlag 获取最后一个标志
func GetLastFlag(chain uint64) []byte {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	var reply = make([]byte, 100)
	err = client.Call("TDb.GetLastFlag", &chain, &reply)
	if err != nil {
		log.Println("fail to GetLastFlag:", err)
		client.Close()
		client = nil
		return nil
	}
	return reply
}

// Commit 提交，将数据写入磁盘，标志清除
func Commit(chain uint64, flag []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := FlagArgs{chain, flag}
	var reply bool
	err = client.Call("TDb.CommitFlag", &args, &reply)
	if err != nil {
		log.Println("fail to CommitFlag:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// Cancel 取消提交，将数据回滚
func Cancel(chain uint64, flag []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := FlagArgs{chain, flag}
	var reply bool
	err = client.Call("TDb.CancelFlag", &args, &reply)
	if err != nil {
		//log.Println("fail to CancelFlag:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// Rollback 将指定标志之后的所有操作回滚，要求当前没有开启标志
func Rollback(chain uint64, flag []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := FlagArgs{chain, flag}
	var reply bool
	err = client.Call("TDb.Rollback", &args, &reply)
	if err != nil {
		log.Println("fail to Rollback:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// Set 存储数据，不携带标签，不会被回滚,tbName的put一值不用flag，否则可能导致数据混乱
func Set(chain uint64, tbName, key, value []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := SetArgs{chain, tbName, key, value}
	var reply bool
	err = client.Call("TDb.Set", &args, &reply)
	if err != nil {
		log.Println("fail to TDb.Set:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// SetWithFlag 写入数据，标志仅仅是一个标志，方便数据回滚
// 每个flag都有对应的historyDb文件，用于记录tbName.key的前一个标签记录位置
// 同时记录本标签最终设置的值，方便回滚
func SetWithFlag(chain uint64, flag, tbName, key, value []byte) error {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := SetWithFlagArgs{chain, flag, tbName, key, value}
	var reply bool
	err = client.Call("TDb.SetWithFlag", &args, &reply)
	if err != nil {
		log.Println("fail to TDb.SetWithFlag:", err)
		client.Close()
		client = nil
		return err
	}

	return err
}

// Get 获取数据
func Get(chain uint64, tbName, key []byte) []byte {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := GetArgs{chain, tbName, key}
	var reply = make([]byte, 65536)
	err = client.Call("TDb.Get", &args, &reply)
	if err != nil {
		log.Println("fail to TDb.Get:", err)
		client.Close()
		client = nil
		return nil
	}

	return reply
}

// Exist 数据是否存在
func Exist(chain uint64, tbName, key []byte) bool {
	var err error
	lock <- 1
	defer func() { <-lock }()
	if client == nil {
		client, err = rpc.DialHTTP(addrType, dbServer)
	}
	// client, err := rpc.DialHTTP(addrType, dbServer)
	// defer client.Close()

	args := GetArgs{chain, tbName, key}
	var reply bool
	err = client.Call("TDb.Exist", &args, &reply)
	if err != nil {
		log.Println("fail to TDb.Exist:", err)
		client.Close()
		client = nil
		return false
	}

	return reply
}
