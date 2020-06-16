package conf

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/govm-net/govm/wallet"
)

// TConfig config of app
type TConfig struct {
	ServerHost      string `json:"server_host,omitempty"`
	HTTPAddress     string `json:"http_address,omitempty"`
	HTTPPort        int    `json:"http_port,omitempty"`
	DbAddrType      string `json:"db_addr_type,omitempty"`
	DbServerAddr    string `json:"db_server_addr,omitempty"`
	CorePackName    []byte `json:"core_pack_name,omitempty"`
	WalletAddr      []byte `json:"wallet_addr,omitempty"`
	SignPrefix      []byte `json:"sign_prefix,omitempty"`
	PrivateKey      []byte `json:"private_key,omitempty"`
	Password        string `json:"password,omitempty"`
	WalletFile      string `json:"wallet_file,omitempty"`
	SaveLog         bool   `json:"save_log,omitempty"`
	IdentifyingCode bool   `json:"identifying_code,omitempty"`
	TrustedServer   string `json:"trusted_server,omitempty"`
	CheckBlock      bool   `json:"check_block,omitempty"`
	AutoRollback    bool   `json:"auto_rollback,omitempty"`
	SaveNodeInfo    bool   `json:"save_node_info,omitempty"`
	NetID           string `json:"net_id,omitempty"`
	OneConnPerMiner bool   `json:"one_conn_per_miner,omitempty"`
	MinerConnLimit  int    `json:"miner_conn_limit,omitempty"`
	VerifyNetData   bool   `json:"verify_net_data,omitempty"`
	SafeEnvironment bool   `json:"safe_environment,omitempty"`
	PProfAddr       string `json:"pprof_addr,omitempty"`
}

var (
	conf TConfig
	// Version software version
	Version string = "v0.5.5"
	// BuildTime build time
	BuildTime string
	// GitHead git head
	GitHead string
)

func init() {
	bit := 32 << (^uint(0) >> 63)
	if bit == 32 {
		log.Println("must be 64 bit system")
		os.Exit(2)
	}
	err := loadConfig()
	if err != nil {
		log.Println("fail to read file,conf.json,", err)
		os.Exit(2)
	}
	log.Printf("software version:%s,build time:%s,git head:%s", Version, BuildTime, GitHead)
}

func loadConfig() error {
	conf.CheckBlock = true
	conf.AutoRollback = true
	conf.VerifyNetData = true
	fn := "./conf/conf.json"
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		data, _ := ioutil.ReadFile("./conf/conf.bak.json")
		if len(data) > 0 {
			ioutil.WriteFile(fn, data, 0666)
		}
	}
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Println("fail to read file,conf.json")
		return err
	}
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Println("fail to Unmarshal configure,conf.json")
		return err
	}
	//log.Println("config info:", conf)
	conf.CorePackName, _ = hex.DecodeString("ff0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")

	if conf.WalletFile == "" {
		conf.WalletFile = "./conf/wallet.key"
	}
	if conf.Password == "" {
		conf.Password = "govm_pwd@2019"
	}
	if conf.HTTPAddress == "" {
		conf.HTTPAddress = "127.0.0.1"
	}
	if conf.TrustedServer == "" {
		conf.TrustedServer = "http://govm.net:9090"
	}

	return nil
}

// GetConf get configure
func GetConf() TConfig {
	return conf
}

// Reload reload config
func Reload() error {
	old := conf
	err := loadConfig()
	if err != nil {
		log.Println("fail to reload config:", err)
		conf = old
	}
	return err
}

// LoadWallet load wallet
func LoadWallet(fileName, password string) {
	w, err := wallet.LoadWallet(fileName, password)
	if err != nil {
		if _, exist := os.Stat(fileName); !os.IsNotExist(exist) {
			fmt.Println("fail to load wallet.", err)
			os.Exit(4)
		}
		os.Rename(fileName, fileName+".error")
		w, err = wallet.LoadWallet("./conf/base.dat", password)
		if err != nil {
			w.Key = wallet.NewPrivateKey()
			pubKey := wallet.GetPublicKey(w.Key)
			w.Address = wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
		} else {
			now := time.Now().Unix() * 1000
			w.Key, w.SignPrefix = wallet.NewChildPrivateKeyOfIBS(w.Key, uint64(now))
		}
		wallet.SaveWallet(fileName, password, w.Address, w.Key, w.SignPrefix)
	}
	conf.PrivateKey = w.Key
	conf.WalletAddr = w.Address
	conf.SignPrefix = w.SignPrefix
}
