package conf

import (
	"encoding/hex"
	"encoding/json"
	"github.com/lengzhao/govm/wallet"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// TConfig config of app
type TConfig struct {
	ServerHost        string `json:"server_host,omitempty"`
	HTTPPort          int    `json:"http_port,omitempty"`
	DbAddrType        string `json:"db_addr_type,omitempty"`
	DbServerAddr      string `json:"db_server_addr,omitempty"`
	CorePackName      []byte `json:"core_pack_name,omitempty"`
	FirstTransName    []byte `json:"first_trans_name,omitempty"`
	ChainOfMine       uint64 `json:"chain_of_mine,omitempty"`
	EnergyOfTrans     uint64 `json:"energy_of_trans,omitempty"`
	EnergyLimitOfMine uint64 `json:"energy_limit_of_mine,omitempty"`
	WalletAddr        []byte `json:"wallet_addr,omitempty"`
	SignPrefix        []byte `json:"sign_prefix,omitempty"`
	PrivateKey        []byte `json:"private_key,omitempty"`
	Password          string `json:"password,omitempty"`
	WalletFile        string `json:"wallet_file,omitempty"`
	CostOfRegMiner    uint64 `json:"cost_of_reg_miner,omitempty"`
	DoMine            bool   `json:"do_mine,omitempty"`
	SaveLog           bool   `json:"save_log,omitempty"`
}

var conf TConfig

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
}

func loadConfig() error {
	data, err := ioutil.ReadFile("./conf/conf.json")
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
	conf.FirstTransName, _ = hex.DecodeString("4c8189d591682b524ea58e61447a3ed8734774dc30ff77e36d563f8a7868cc86")
	conf.CorePackName, _ = hex.DecodeString("9edcee1a25950643c09476b7c039eb8aec09141a8d0e80051fd52a0e37bc60fe")

	if conf.WalletFile == "" {
		conf.WalletFile = "./conf/wallet.key"
	}
	if conf.Password == "" {
		conf.Password = "govm_pwd_2019"
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
