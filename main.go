package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
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
	conf.LoadWallet(c.WalletFile, c.Password)
	// start database server
	{
		db := database.TDb{}
		db.Init()
		defer db.Close()
		sr := rpc.NewServer()
		sr.Register(&db)
		sr.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

		server := &http.Server{
			Addr:         c.DbServerAddr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      sr,
		}
		go server.ListenAndServe()
	}

	// startHTTPServer
	{
		addr := fmt.Sprintf("localhost:%d", c.HTTPPort)
		router := api.NewRouter()
		go http.ListenAndServe(addr, router)
	}
	n := network.New()
	if n == nil {
		log.Println("error address")
		os.Exit(2)
	}

	{
		data, err := ioutil.ReadFile("./conf/bootstrap.json")
		if err == nil {
			var peers []string
			err = json.Unmarshal(data, &peers)
			if err == nil {
				n.RegistPlugin(plugins.NewBootstrap(peers))
			}
		}
	}

	n.RegistPlugin(new(plugins.DiscoveryPlugin))
	n.RegistPlugin(plugins.NewBroadcast(0))
	key := loadNodeKey()
	cp := crypto.NewMgr()
	cp.Register(new(wallet.EcdsaKey))
	cp.SetPrivKey("ecdsa", key)
	n.SetKeyMgr(cp)
	n.RegistPlugin(new(handler.MsgPlugin))
	n.RegistPlugin(new(handler.InternalPlugin))

	go n.Listen(c.ServerHost)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)

	for {
		select {
		case <-sig:
			break
		case <-api.StopFlag:
			break
		}
	}
	n.Close()
	handler.GraceStop()
}

func loadNodeKey() []byte {
	const (
		nodeKeyFile = "./conf/node_key.dat"
		passwd      = "10293847561029384756"
	)
	_, privKey, _ := wallet.LoadWallet(nodeKeyFile, passwd)
	if privKey == nil {
		privKey = wallet.NewPrivateKey()
		pubKey := wallet.GetPublicKey(privKey)
		addr := wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
		wallet.SaveWallet(nodeKeyFile, passwd, addr, privKey, nil)
	}
	return privKey
}
