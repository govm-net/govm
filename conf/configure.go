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
	conf.FirstTransName, _ = hex.DecodeString("ef2dc3bb54242ba576542cb17af4175124c0e443ad2c794cb753e0ad11b23f73")
	conf.CorePackName, _ = hex.DecodeString("365d2b302434dac708688612b3b86a486d59c01071be7b2738eb8c6c028fd413")

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
