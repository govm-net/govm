package wallet

import (
	"encoding/hex"
	"log"
	"testing"
)

func TestWallet(t *testing.T) {
	priv := NewPrivateKey()

	pubK := GetPublicKey(priv)
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
	parentPubK := GetPublicKey(parentPriv)
	address := PublicKeyToAddress(parentPubK, EAddrTypeIBS)
	log.Println("address:", hex.EncodeToString(address))
	log.Println("parentPubK:", hex.EncodeToString(parentPubK))
	msg := []byte("11111111,this is the message")
	tm := bytesToUint64(msg)
	childPriv, signPri := NewChildPrivateKeyOfIBS(parentPriv, tm)
	childPubK := GetPublicKey(childPriv)
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
