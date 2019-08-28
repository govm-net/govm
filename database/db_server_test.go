package database

import (
	"bytes"
	"net/http"
	"net/rpc"
	"testing"
	"time"

	"github.com/lengzhao/govm/conf"
)

func TestTDbServer(t *testing.T) {
	c := conf.GetConf()
	db := new(TDb)
	db.Init()
	defer db.Close()
	sr := rpc.NewServer()
	sr.Register(db)
	sr.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	server := &http.Server{Addr: c.DbServerAddr, Handler: sr}
	go func(server *http.Server) {
		server.ListenAndServe()
	}(server)

	time.Sleep(2 * time.Second)
	var tbName = []byte("table")
	var key = []byte("key")
	var value = []byte("value")
	for i := 0; i < 1000; i++ {
		err := Set(1, tbName, key, value)
		if err != nil {
			t.Fatal(err)
		}
		v := Get(1, tbName, key)
		if bytes.Compare(value, v) != 0 {
			t.Errorf("different value.hope:%x,get:%x\n", value, v)
		}
	}
	server.Close()
}
