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
	ForceMine         bool   `json:"force_mine,omitempty"`
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
	conf.FirstTransName, _ = hex.DecodeString("0e79d2d6b8bf0ea6b1b48ebf7a0a95adc6f92e5b05393e66340861fd52b92fcc")
	conf.CorePackName, _ = hex.DecodeString("cb2fb3994c274446f5dd4d8397d2f73ad68f32f649e2577c23877f3a4d7e1a05")

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
	addr, privKey, prefix := wallet.LoadWallet(fileName, password)
	if addr == nil {
		addr, privKey, prefix = wallet.LoadWallet("./conf/base.dat", password)
		if addr == nil {
			privKey = wallet.NewPrivateKey()
			pubKey := wallet.GetPublicKey(privKey)
			addr = wallet.PublicKeyToAddress(pubKey, wallet.EAddrTypeDefault)
		} else {
			now := time.Now().Unix() * 1000
			privKey, prefix = wallet.NewChildPrivateKeyOfIBS(privKey, uint64(now))
		}
		wallet.SaveWallet(fileName, password, addr, privKey, prefix)
	}
	conf.PrivateKey = privKey
	conf.WalletAddr = addr
	conf.SignPrefix = prefix
}
