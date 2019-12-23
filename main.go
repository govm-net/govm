package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/lengzhao/govm/api"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/handler"
	"github.com/lengzhao/govm/wallet"
	"github.com/lengzhao/libp2p/crypto"
	"github.com/lengzhao/libp2p/network"
	"github.com/lengzhao/libp2p/plugins"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	c := conf.GetConf()
	if c.SaveLog {
		lf := new(logWriter)
		defer lf.Close()
		log.SetOutput(lf)
	}

	conf.LoadWallet(c.WalletFile, c.Password)
	// start database server
	{
		db := database.TDb{}
		db.Init()
		defer db.Close()
		sr := rpc.NewServer()
		sr.Register(&db)
		sr.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

		ln, err := net.Listen(c.DbAddrType, c.DbServerAddr)
		if err != nil {
			log.Println("fail to start db Listen:", c.DbServerAddr, err)
			return
		}

		server := &http.Server{
			Addr: c.DbServerAddr,
			// ReadTimeout:  10 * time.Second,
			// WriteTimeout: 10 * time.Second,
			// IdleTimeout:  20 * time.Second,
			Handler: sr,
		}
		// go server.ListenAndServe()

		go server.Serve(ln)
	}

	// startHTTPServer
	{
		addr := fmt.Sprintf("localhost:%d", c.HTTPPort)
		router := api.NewRouter()
		go func() {
			err := http.ListenAndServe(addr, router)
			if err != nil {
				log.Println("fail to http Listen:", addr, err)
				os.Exit(2)
			}
		}()
	}
	n := network.New()
	if n == nil {
		log.Println("fail to new network")
		os.Exit(2)
	}

	{
		data, err := ioutil.ReadFile("./conf/bootstrap.json")
		if err == nil {
			var peers []string
			err = json.Unmarshal(data, &peers)
			if err == nil {
				b := new(plugins.Bootstrap)
				b.Addrs = peers
				n.RegistPlugin(b)
			}
		}
	}

	n.RegistPlugin(new(plugins.DiscoveryPlugin))
	n.RegistPlugin(new(plugins.Broadcast))
	key := loadNodeKey()
	cp := crypto.NewMgr()
	cp.Register(new(wallet.EcdsaKey))
	cp.SetPrivKey("ecdsa", key)
	n.SetKeyMgr(cp)
	n.RegistPlugin(new(handler.MsgPlugin))
	n.RegistPlugin(new(handler.InternalPlugin))
	n.RegistPlugin(new(handler.SyncPlugin))
	n.RegistPlugin(new(handler.NATTPlugin))

	err := n.Listen(c.ServerHost)
	if err != nil {
		log.Fatal("fail to listen:", c.ServerHost, err)
	}
	n.Close()
	log.Println("wait to exit(5s)")
	time.Sleep(5 * time.Second)
	handler.Exit()
}

func loadNodeKey() []byte {
	const (
		nodeKeyFile = "./conf/node_key.dat"
		passwd      = "10293847561029384756"
	)
	w, err := wallet.LoadWallet(nodeKeyFile, passwd)
	if err != nil {
		w.Key = wallet.NewPrivateKey()
		pubKey := wallet.GetPublicKey(w.Key)
		addr := wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
		wallet.SaveWallet(nodeKeyFile, passwd, addr, w.Key, nil)
	}
	return w.Key
}

type logWriter struct {
	os.File
	size int
	fn   string
	f    *os.File
}

func (l *logWriter) Write(data []byte) (int, error) {
	now := time.Now()
	fn := fmt.Sprintf("./log/govm_%d%02d%02d.log", now.Year(), now.Month(), now.Day())
	if fn != l.fn {
		old := now.Add(-time.Hour * 24 * 5)
		ofn := fmt.Sprintf("./log/govm_%d%02d%02d.log", old.Year(), old.Month(), old.Day())
		os.Remove(ofn)
		l.fn = fn
		if l.f == nil {
			os.Mkdir("./log/", 0755)
		} else {
			l.f.Close()
			l.f = nil
		}
		l.size = 0
		file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
		if err != nil {
			return 0, err
		}
		l.f = file
		os.Stdout = file
		os.Stderr = file
	}
	if l.f == nil {
		return 0, nil
	}
	if l.size > (500 << 20) {
		l.f.Seek(0, 0)
		l.size = 0
	}
	l.size += len(data)
	return l.f.Write(data)
}

func (l *logWriter) Close() {
	if l.f != nil {
		l.f.Close()
	}
}
