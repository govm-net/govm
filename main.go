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
	lf := new(logWriter)
	defer lf.Close()
	log.SetOutput(lf)
	

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
		l.fn = fn
		if l.f == nil {
			os.Mkdir("./log/", 766)
		}else{
			l.f.Close()
			l.f = nil
		}
		l.size = 0
		file, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_SYNC, 0755)
		if err != nil {
			return 0, err
		}
		l.f = file
		os.Stdout = file
		os.Stderr = file
	}
	if l.f == nil{
		return 0,nil
	}
	if l.size > (500<<20) {
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
