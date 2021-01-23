package wallet

import (
	"bytes"
	"encoding/hex"
	"log"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/sha3"
)

func TestWallet(t *testing.T) {
	priv := NewPrivateKey()

	pubK := GetPublicKey(priv, EAddrTypeDefault)
	address := PublicKeyToAddress(pubK, EAddrTypeDefault)
	msg := []byte("1234566")
	sign := Sign(priv, msg)
	log.Println("sign:", len(sign), hex.EncodeToString(sign))
	recov := Recover(address, sign, msg)

	log.Println("public key:", hex.EncodeToString(pubK[:]))
	log.Println("address:", hex.EncodeToString(address[:]))
	log.Println("recov:", recov)
	if !recov {
		t.Error("recover error")
	}
}

func TestNewChildPrivateKeyOfIBS(t *testing.T) {
	parentPriv := NewPrivateKey()
	// parentPriv := getHash([]byte("aaaaa"))
	parentPubK := GetPublicKey(parentPriv, EAddrTypeDefault)
	address := PublicKeyToAddress(parentPubK, EAddrTypeIBS)
	log.Println("address:", hex.EncodeToString(address))
	log.Println("parentPubK:", hex.EncodeToString(parentPubK))
	msg := []byte("11111111,this is the message")
	tm := bytesToUint64(msg)
	childPriv, signPri := NewChildPrivateKeyOfIBS(parentPriv, tm)
	childPubK := GetPublicKey(childPriv, EAddrTypeDefault)
	log.Println("childPubK:", hex.EncodeToString(childPubK))
	signSuf := Sign(childPriv, msg)
	log.Println("signPri:", hex.EncodeToString(signPri))
	log.Println("signSuf:", hex.EncodeToString(signSuf))
	sign := append(signPri, signSuf...)
	recov := Recover(address, sign, msg)
	if !recov {
		t.Error("recover error")
	}
	// t.Error("aa")
}

func TestRand(t *testing.T) {
	priv := GetHash([]byte("aaaaa"))
	msg := []byte("11111111,this is the message")
	signSuf := Sign(priv, msg)
	log.Println("signSuf:", hex.EncodeToString(signSuf))
	// t.Error("aa")
}

func TestSha256(t *testing.T) {
	hashPrefix = nil
	h1 := GetHash([]byte("1"))
	h2 := GetHash([]byte("12"))
	h3 := GetHash([]byte("123"))
	h4 := GetHash([]byte("1234567890123456789012345678901234567890123456789012345678901234567890"))
	h5 := GetHash([]byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"))
	log.Printf("\nh1:%x\nh2:%x\nh3:%x\nh4:%x\nh5:%x\n", h1, h2, h3, h4, h5)
	// log.Printf("\nh1:%x\nh5:%x\n", h1, h5)
}

func TestNewPrivate(t *testing.T) {
	priKey, _ := btcec.NewPrivateKey(btcec.S256())
	out := priKey.Serialize()
	log.Printf("private key:%x\n", out)
}

func TestSign001(t *testing.T) {
	msg := "00000001e7c18b902c83a33f53cf8ec64cf71b93568ab7c427bef126ed0a9cfa"
	data, _ := hex.DecodeString(msg)
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), data)
	signature, _ := btcec.SignCompact(btcec.S256(), privKey, data, true)
	t.Errorf("sign:%x\n", signature)
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), data)
	t.Errorf("pubKey1:%x\n", pubKey.SerializeCompressed())
	t.Errorf("pubKey2:%x\n", pubKey.SerializeUncompressed())
	sig, _ := hex.DecodeString("1cdcadfd3168ab3ba2d00439b95e5d2ff9aa418bc5873061f558bf24224e431d24235202ce9754c45d2ffbc646a1a2d005106d512128183e4667133a6a81f3241c")
	pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sig, data)
	t.Errorf("recover\npk:%x\ncompress:%v\nerr:%s\n", pk.SerializeCompressed(), wasCompressed, err)
}

func TestEthAddress(t *testing.T) {
	pubKey := "d061e9c5891f579fd548cfd22ff29f5c642714cc7e7a9215f0071ef5a5723f691757b28e31be71f09f24673eed52348e58d53bcfd26f4d96ec6bf1489eab429d"
	k, _ := hex.DecodeString(pubKey)
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(k)
	rst := keccak.Sum(nil)
	t.Errorf("addr:%x\n", rst[12:])
}

func TestEthAddress2(t *testing.T) {
	// priv := NewPrivateKey()
	str := "1111111111111111111111111111111111111111111111111111111111111111"
	priv, _ := hex.DecodeString(str)
	_, k2 := btcec.PrivKeyFromBytes(btcec.S256(), priv)
	pubKey := k2.SerializeUncompressed()

	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(pubKey[1:])
	rst := keccak.Sum(nil)
	address1 := rst[12:]

	pubK := GetPublicKey(priv, EAddrEthereum)
	address := PublicKeyToAddress(pubK, EAddrEthereum)
	if bytes.Compare(address1, address[4:]) != 0 {
		t.Errorf("addr1:%x,addr2:%x", address1, address)
	}
}
