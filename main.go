package main

import (
	"encoding/json"
	"fmt"
	"github.com/lengzhao/govm/api"
	"github.com/lengzhao/govm/conf"
	"github.com/lengzhao/govm/database"
	"github.com/lengzhao/govm/handler"
	"github.com/lengzhao/govm/wallet"
	"github.com/lengzhao/libp2p/crypto"
	"github.com/lengzhao/libp2p/network"
	"github.com/lengzhao/libp2p/plugins"
	"gopkg.in/natefinch/lumberjack.v2"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	c := conf.GetConf()
	if c.SaveLog {
		log.SetOutput(&lumberjack.Logger{
			Filename:   "./log/govm.log",
			MaxSize:    500, // megabytes
			MaxBackups: 5,
			MaxAge:     10,   //days
			Compress:   true, // disabled by default
		})
	}

	conf.LoadWallet(c.WalletFile, c.Password)
	// start database server
	if !c.SeparateDB {
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
		log.Println("fail to listen:", c.ServerHost, err)
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
