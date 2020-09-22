package zff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/lengzhao/database/client"
	"github.com/lengzhao/database/server"
)

type memKey struct {
	TbName string
	Key    string
}

// dBNWProxy proxy of database, not write data to database
type dBNWProxy struct {
	chain uint64
	mu    sync.Mutex
	flag  []byte
	cache map[memKey][]byte
	dbc   *client.Client
}

type connItem struct {
	l       net.Listener
	address string
	c       *client.Client
}

// DBNWProxyMgr db server, proxy and cache, not write data to database
type DBNWProxyMgr struct {
	mu        sync.Mutex
	freeConns chan *connItem
	useConns  map[string]*connItem
	addType   string
}

var dfMgr *DBNWProxyMgr

// InitDefaultDbMgr init default db proxy, not write data to database
func InitDefaultDbMgr(addrType, address string, number int) {
	dfMgr = NewDbProxyMgr(addrType, address, number)
}

// NewDbProxyMgr new db proxy manager
func NewDbProxyMgr(addrType, address string, number int) *DBNWProxyMgr {
	mgr := DBNWProxyMgr{}
	mgr.freeConns = make(chan *connItem, number)
	mgr.useConns = make(map[string]*connItem)
	mgr.addType = addrType

	for i := 0; i < number; i++ {
		db := server.NewRPCObj(".")
		server.RegisterAPI(db, func(dir string, id uint64) server.DBApi {
			return newProxy(id, addrType, address)
		})
		rServer := rpc.NewServer()
		rServer.Register(db)
		// rServer.HandleHTTP()
		hsrv := http.NewServeMux()
		hsrv.Handle(rpc.DefaultRPCPath, rServer)
		lis, err := net.Listen(addrType, "127.0.0.1:0")
		if err != nil {
			log.Fatalln("fatal error: ", err)
		}
		srv := &http.Server{
			IdleTimeout: 120 * time.Second,
			Handler:     hsrv,
		}
		go srv.Serve(lis)
		item := connItem{}
		item.l = lis
		item.address = lis.Addr().String()
		log.Println("start one proxy server.", addrType, item.address)
		mgr.freeConns <- &item
	}

	return &mgr
}

func (m *DBNWProxyMgr) alloc() (c *client.Client) {
	select {
	case item := <-m.freeConns:
		if item.c == nil {
			item.c = client.New(m.addType, item.address, 1)
		}
		if item.c == nil {
			m.freeConns <- item
			return nil
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.useConns[item.address] = item
		return item.c
	case <-time.After(time.Second * 5):
		log.Println("warning.DBNWProxyMgr alloc timeout.")
		return nil
	}
}

func (m *DBNWProxyMgr) free(c *client.Client) {
	_, addr := c.GetAddress()
	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.useConns[addr]
	delete(m.useConns, addr)
	m.freeConns <- item
}

// NewProxy new proxy of database
func newProxy(chain uint64, addType, address string) server.DBApi {
	out := new(dBNWProxy)
	out.chain = chain
	out.cache = make(map[memKey][]byte)
	out.dbc = client.New(addType, address, 1)
	return out
}

// Close close
func (db *dBNWProxy) Close() {
	db.dbc.Close()
}

// OpenFlag open flag
func (db *dBNWProxy) OpenFlag(flag []byte) error {
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
func (db *dBNWProxy) GetLastFlag() []byte {
	return db.flag
}

// Commit not write, only reset cache
func (db *dBNWProxy) Commit(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// Cancel only reset cache
func (db *dBNWProxy) Cancel(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// Rollback only reset cache
func (db *dBNWProxy) Rollback(flag []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache = make(map[memKey][]byte)
	db.flag = nil
	return nil
}

// SetWithFlag set with flag, only write to cache
func (db *dBNWProxy) SetWithFlag(flag, tbName, key, value []byte) error {
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
func (db *dBNWProxy) Set(tbName, key, value []byte) error {
	mk := memKey{}
	mk.TbName = hex.EncodeToString(tbName)
	mk.Key = hex.EncodeToString(key)
	db.mu.Lock()
	defer db.mu.Unlock()
	db.cache[mk] = value
	return nil
}

// Get get from cache,if not exist, read from database server
func (db *dBNWProxy) Get(tbName, key []byte) []byte {
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
func (db *dBNWProxy) Exist(tbName, key []byte) bool {
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
func (db *dBNWProxy) GetNextKey(tbName, preKey []byte) []byte {
	return db.dbc.GetNextKey(db.chain, tbName, preKey)
}
