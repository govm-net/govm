package a1000000000000000000000000000000000000000000000000000000000000000

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"testing"

	core "github.com/govm-net/govm/tools/debugger/zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	"github.com/lengzhao/database/client"
	"github.com/lengzhao/database/server"
)

var blockKey = [32]byte{1, 2, 3, 4, 5}
var chain uint64 = 1
var conf Config

func TestMain(m *testing.M) {
	data, err := ioutil.ReadFile("./conf/conf.json")
	if err != nil {
		fmt.Println("fail to load conf.", err)
		os.Exit(2)
	}
	json.Unmarshal(data, &conf)

	db := server.NewRPCObj(".")
	server.RegisterAPI(db, func(dir string, id uint64) server.DBApi {
		return NewProxy(id)
	})

	rpc.Register(db)
	rpc.HandleHTTP()
	lis, err := net.Listen(conf.DbAddrType, conf.DbServerAddr)
	if err != nil {
		log.Fatalln("fatal error: ", err)
	}
	go http.Serve(lis, nil)
	m.Run()
}

// Config config of test
type Config struct {
	DbAddrType      string `json:"db_addr_type,omitempty"`
	DbServerAddr    string `json:"db_server_addr,omitempty"`
	RelDbAddrType   string `json:"rel_db_addr_type,omitempty"`
	RelDbServerAddr string `json:"rel_db_server_addr,omitempty"`
}

type memKey struct {
	TbName string
	Key    string
}

// DBNWProxy proxy of database, not write data to database
type DBNWProxy struct {
	chain uint64
	mu    sync.Mutex
	flag  []byte
	cache map[memKey][]byte
	dbc   *client.Client
}

// NewProxy new proxy of database
func NewProxy(chain uint64) server.DBApi {
	out := new(DBNWProxy)
	out.chain = chain
	out.cache = make(map[memKey][]byte)
	out.dbc = client.New(conf.RelDbAddrType, conf.RelDbServerAddr, 2)
	return out
}

// Close close
func (db *DBNWProxy) Close() {
	db.dbc.Close()
}

// OpenFlag open flag
func (db *DBNWProxy) OpenFlag(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(db.flag) != 0 {
		log.Println("fail to open flag, exist flag")
		return fmt.Errorf("exist flag")
	}
	db.flag = flag
	db.cache = make(map[memKey][]byte)
	return nil
}

// GetLastFlag return opened flag or nil
func (db *DBNWProxy) GetLastFlag() []byte {
	return db.flag
}

// Commit not write, only reset cache
func (db *DBNWProxy) Commit(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// Cancel only reset cache
func (db *DBNWProxy) Cancel(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// Rollback only reset cache
func (db *DBNWProxy) Rollback(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// SetWithFlag set with flag, only write to cache
func (db *DBNWProxy) SetWithFlag(flag, tbName, key, value []byte) error {
	if bytes.Compare(db.flag, flag) != 0 {
		return fmt.Errorf("different flag")
	}
	mk := memKey{}
	mk.TbName = hex.EncodeToString(tbName)
	mk.Key = hex.EncodeToString(key)
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache[mk] = value
	return nil
}

// Set only write to cache
func (db *DBNWProxy) Set(tbName, key, value []byte) error {
	mk := memKey{}
	mk.TbName = hex.EncodeToString(tbName)
	mk.Key = hex.EncodeToString(key)
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache[mk] = value
	return nil
}

// Get get from cache,if not exist, read from database server
func (db *DBNWProxy) Get(tbName, key []byte) []byte {
	mk := memKey{}
	mk.TbName = hex.EncodeToString(tbName)
	mk.Key = hex.EncodeToString(key)
	db.mu.Lock()
	v, ok := db.cache[mk]
	db.mu.Unlock()
	if ok {
		return v
	}
	return db.dbc.Get(db.chain, tbName, key)
}

// Exist if exist return true
func (db *DBNWProxy) Exist(tbName, key []byte) bool {
	mk := memKey{}
	mk.TbName = hex.EncodeToString(tbName)
	mk.Key = hex.EncodeToString(key)
	db.mu.Lock()
	_, ok := db.cache[mk]
	db.mu.Unlock()
	if ok {
		return true
	}
	return db.dbc.Exist(db.chain, tbName, key)
}

// GetNextKey get next key for visit
func (db *DBNWProxy) GetNextKey(tbName, preKey []byte) []byte {
	return db.dbc.GetNextKey(db.chain, tbName, preKey)
}

func hexToBytes(in string) []byte {
	out, err := hex.DecodeString(in)
	if err != nil {
		fmt.Println("fail to decode hex:", err)
		panic(err)
	}
	return out
}

func Test_run(t *testing.T) {
	fmt.Println("testing start", t.Name())
	core.ChainID = 1
	core.InitForTest()
	core.SetAppAccountForTest(tApp{}, 1<<45)
	run(hexToBytes("02984010319cd34659f7fcb20b31d615d850ab32ca930618"), []byte("parament"), 10)
}
