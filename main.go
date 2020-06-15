package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/govm-net/govm/api"
	"github.com/govm-net/govm/conf"
	"github.com/govm-net/govm/database"
	"github.com/govm-net/govm/handler"
	"github.com/govm-net/govm/wallet"
	"github.com/lengzhao/libp2p/crypto"
	"github.com/lengzhao/libp2p/network"
	"github.com/lengzhao/libp2p/plugins"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	c := conf.GetConf()
	if c.SaveLog {
		log.SetOutput(&lumberjack.Logger{
			Filename:   "./log/govm.log",
			MaxSize:    50, // megabytes
			MaxBackups: 5,
			MaxAge:     10,   //days
			Compress:   true, // disabled by default
		})
	} else {
		log.SetOutput(ioutil.Discard)
	}
	database.ChangeClientNumber(10)
	client := database.GetClient()
	val := client.Get(1, []byte("info"), []byte("net"))
	if len(val) == 0 {
		err := client.Set(1, []byte("info"), []byte("net"), []byte(c.NetID))
		if err != nil {
			fmt.Println("fail to set database,make sure the database server running.", err)
			os.Exit(2)
		}
	} else if string(val) != c.NetID {
		fmt.Println("different net id,hope:", c.NetID, ", get:", val)
		os.Exit(3)
	}

	conf.LoadWallet(c.WalletFile, c.Password)
	// startHTTPServer
	{
		addr := fmt.Sprintf("%s:%d", c.HTTPAddress, c.HTTPPort)
		router := api.NewRouter()
		go func() {
			err := http.ListenAndServe(addr, router)
			if err != nil {
				fmt.Println("fail to http Listen:", addr, err)
				os.Exit(2)
			}
		}()
		if c.PProfAddr != "" {
			go http.ListenAndServe(c.PProfAddr, nil)
		}
	}
	n := network.New()
	if n == nil {
		fmt.Println("fail to new network")
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
	rk := wallet.EcdsaKey{Type: c.NetID, NeedVerify: c.VerifyNetData}
	cp := crypto.NewMgr()
	cp.Register(&rk)
	cp.SetPrivKey(rk.GetType(), key)
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
	log.Println("wait to exit(3s)")
	time.Sleep(3 * time.Second)
}

func loadNodeKey() []byte {
	if !conf.GetConf().SaveNodeInfo {
		return wallet.NewPrivateKey()
	}
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
