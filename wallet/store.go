package wallet

import (
	"encoding/hex"
	"encoding/json"
	"github.com/govm-net/govm/encrypt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// TWallet 钱包存储的结构体
type TWallet struct {
	Tag        string `json:"tag,omitempty"`
	AddressStr string `json:"address_str,omitempty"`
	Address    []byte `json:"address,omitempty"`
	Key        []byte `json:"key,omitempty"`
	SignPrefix []byte `json:"sign_prefix,omitempty"`
}

// LoadWallet 从文件中加载钱包信息
func LoadWallet(file, password string) (TWallet, error) {
	info := TWallet{}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println("fail to read config file:", file, err)
		return info, err
	}

	err = json.Unmarshal(data, &info)
	if err != nil {
		log.Println("fail to json.Unmarshal:", err)
		return info, err
	}

	aesEnc := encrypt.AesEncrypt{}
	aesEnc.Key = password
	privKey, err := aesEnc.Decrypt(info.Key)
	if err != nil {
		log.Println("fail to decrypt privateKey.", err)
		return info, err
	}
	info.Key = privKey

	return info, nil
}

// SaveWallet 将私钥信息和地址保存到文件中
func SaveWallet(file, passwd string, addr, privKey, prefix []byte) error {
	aesEnc := encrypt.AesEncrypt{}
	aesEnc.Key = passwd

	arrEncrypt, err := aesEnc.Encrypt(privKey)
	if err != nil {
		log.Println(err)
		return err
	}
	info := TWallet{}
	info.AddressStr = hex.EncodeToString(addr)
	info.Address = addr
	info.SignPrefix = prefix
	info.Key = arrEncrypt
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
