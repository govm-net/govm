package database

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/lengzhao/database/client"
	"github.com/lengzhao/govm/conf"
)

// NWProxy proxy for test mode,not write database
type NWProxy struct {
	flag  []byte
	mu    sync.Mutex
	data  map[string][]byte
	db    *client.Client
	chain uint64
}

var dfDB *client.Client

// GetClient get database client
func GetClient() *client.Client {
	if dfDB == nil {
		c := conf.GetConf()
		dfDB = client.New(c.DbAddrType, c.DbServerAddr, 10)
	}
	return dfDB
}

// NewNWProxy new proxy(not write database,for test mode)
func NewNWProxy(chain uint64) *NWProxy {
	out := new(NWProxy)
	out.data = make(map[string][]byte)
	out.db = GetClient()
	out.chain = chain
	return out
}

// Close close
func (p *NWProxy) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data = nil
	p.flag = nil
}

// OpenFlag open flag
func (p *NWProxy) OpenFlag(flag []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.flag) != 0 {
		return fmt.Errorf("exist flag")
	}
	p.flag = flag
	return nil
}

// GetLastFlag get last flag
func (p *NWProxy) GetLastFlag() []byte {
	return p.flag
}

// Commit not write,only reset
func (p *NWProxy) Commit(flag []byte) error {
	return p.Cancel(flag)
}

// Cancel cancel
func (p *NWProxy) Cancel(flag []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data = make(map[string][]byte)
	p.flag = nil
	return nil
}

// Rollback only reset
func (p *NWProxy) Rollback(flag []byte) error {
	return p.Cancel(flag)
}

// SetWithFlag only write map
func (p *NWProxy) SetWithFlag(flag, tbName, key, value []byte) error {
	if bytes.Compare(flag, p.flag) != 0 {
		log.Printf("different flag,tbName:%s,key:%x\n", tbName, key)
		return fmt.Errorf("different flag")
	}
	return p.Set(tbName, key, value)
}

// Set only write map
func (p *NWProxy) Set(tbName, key, value []byte) error {
	lk := fmt.Sprintf("%x_%x", tbName, key)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data[lk] = value
	return nil
}

// Get get data
func (p *NWProxy) Get(tbName, key []byte) []byte {
	lk := fmt.Sprintf("%x_%x", tbName, key)
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.data[lk]
	if ok {
		return v
	}
	return p.db.Get(p.chain, tbName, key)
}

// Exist check exist
func (p *NWProxy) Exist(tbName, key []byte) bool {
	lk := fmt.Sprintf("%x_%x", tbName, key)
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.data[lk]
	if ok {
		return true
	}
	return p.db.Exist(p.chain, tbName, key)
}

// GetNextKey only get next from server
func (p *NWProxy) GetNextKey(tbName, preKey []byte) []byte {
	return p.db.GetNextKey(p.chain, tbName, preKey)
}
