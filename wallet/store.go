package wallet

import (
	"encoding/hex"
	"encoding/json"
	"github.com/lengzhao/govm/encrypt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// TWallet 钱包存储的结构体
type TWallet struct {
	Tag        string `json:"tag,omitempty"`
	Address    string `json:"address,omitempty"`
	Key        string `json:"key,omitempty"`
	SignPrefix string `json:"sign_prefix,omitempty"`
}

// LoadWallet 从文件中加载钱包信息
func LoadWallet(file string, password string) (addr, privKey, prefix []byte) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("fail to read config file:", file, err)
		return nil, nil, nil
	}

	info := TWallet{}
	err = json.Unmarshal(data, &info)
	if err != nil {
		log.Println("fail to json.Unmarshal:", err)
		return nil, nil, nil
	}

	addr, _ = hex.DecodeString(info.Address)
	prefix, _ = hex.DecodeString(info.SignPrefix)
	aesEnc := encrypt.AesEncrypt{}
	aesEnc.Key = password
	data, _ = hex.DecodeString(info.Key)
	privKey, err = aesEnc.Decrypt(data)
	if err != nil {
		log.Println("fail to decrypt privateKey", err)
		return nil, nil, nil
	}
	pubKey := GetPublicKey(privKey)
	if len(addr) == 0 {
		addr = PublicKeyToAddress(pubKey, EAddrTypeDefault)
	}

	return
}

// SaveWallet 将私钥信息和地址保存到文件中
func SaveWallet(file string, passwd string, addr, privKey, prefix []byte) error {
	aesEnc := encrypt.AesEncrypt{}
	aesEnc.Key = passwd

	arrEncrypt, err := aesEnc.Encrypt(privKey)
	if err != nil {
		log.Println(err)
		return err
	}
	info := TWallet{}
	info.Address = hex.EncodeToString(addr)
	info.SignPrefix = hex.EncodeToString(prefix)
	info.Key = hex.EncodeToString(arrEncrypt)
	if addr[0] == EAddrTypeIBS {
		t := GetDeadlineOfIBS(addr)
		info.Tag = time.Unix(int64(t/1000), 0).Format(time.RFC3339)
	}

	sd, _ := json.Marshal(info)
	f, err := os.Create(file)
	if err != nil {
		log.Println(err)
		return err
	}
	f.Write(sd)
	f.Close()
	return nil
}
