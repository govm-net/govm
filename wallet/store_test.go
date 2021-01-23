package wallet

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestSaveWallet(t *testing.T) {
	fn1 := "parent.key"
	fn2 := "child.key"
	pwd := "aaaaa"
	// defer os.Remove(fn1)
	defer os.Remove(fn2)
	parentPriv := NewPrivateKey()
	parentPubK := GetPublicKey(parentPriv, EAddrTypeIBS)
	address := PublicKeyToAddress(parentPubK, EAddrTypeIBS)

	tm := uint64(time.Now().Unix() * 1000)
	childPriv, signPri := NewChildPrivateKeyOfIBS(parentPriv, tm)
	SaveWallet(fn1, pwd, address, parentPriv, nil)
	SaveWallet(fn2, pwd, address, childPriv, signPri)

	w1, err := LoadWallet(fn1, pwd)
	if err != nil {
		t.Error("fail to load fn1,", err)
	}
	if bytes.Compare(w1.Key, parentPriv) != 0 {
		t.Error("error key of fn1")
	}
	w2, err := LoadWallet(fn2, pwd)
	if err != nil {
		t.Error("fail to load fn2,", err)
	}
	if bytes.Compare(w2.Key, childPriv) != 0 {
		t.Error("error key of fn2")
	}

	if bytes.Compare(w2.SignPrefix, signPri) != 0 {
		t.Error("error SignPrefix of fn2")
	}
	if bytes.Compare(w1.Address, w2.Address) != 0 {
		t.Error("different Address")
	}

	fmt.Println("address:", w1.Tag, w1.AddressStr, address)

}
