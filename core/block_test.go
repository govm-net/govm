package a9edcee1a25950643c09476b7c039eb8aec09141a8d0e80051fd52a0e37bc60fe

import (
	"github.com/lengzhao/govm/runtime"
	"github.com/lengzhao/govm/wallet"
	"log"
	"testing"
	"time"
)

func TestDecodeBlock(t *testing.T) {
	transList := [2]Hash{}
	transList[0] = Hash{1, 2, 1, 2, 1, 2, 1, 2}
	transList[1] = Hash{2, 1, 2, 1, 2, 1, 2, 1, 2}
	privateKey := runtime.GetHash([]byte("123456"))
	pubKey := wallet.GetPublicKey(privateKey)
	stream := wallet.PublicKeyToAddress(pubKey, 1)
	address := Address{}
	runtime.Decode(stream, &address)
	block := NewBlock(1, address)
	block.Size = 0
	block.Time = 1000000 + maxBlockInterval
	block.SetTransList(transList[:])
	data := block.GetSignData()
	sign := wallet.Sign(privateKey, data)
	block.SetSign(sign)
	data = block.Output()

	b := DecodeBlock(data)
	if b.Key != block.Key {
		t.Errorf("error key:%x,%x", b.Key[:], block.Key[:])
	}
}

func TestMineBlock(t *testing.T) {
	var chain uint64 = 1

	tm := uint64(time.Now().Unix() * 1000)
	parentPriv := wallet.NewPrivateKey()
	parentPubK := wallet.GetPublicKey(parentPriv)
	address := wallet.PublicKeyToAddress(parentPubK, wallet.EAddrTypeIBS)
	childPriv, signPri := wallet.NewChildPrivateKeyOfIBS(parentPriv, tm)

	addr := Address{}
	runtime.Decode(address, &addr)
	block := NewBlock(chain, addr)
	block.Time = tm

	signData := block.GetSignData()
	sign := wallet.Sign(childPriv, signData)
	if len(sign) == 0 {
		t.Error("fail to sign,len(sign)=0")
		return
	}
	if len(signPri) > 0 {
		s := make([]byte, len(signPri))
		copy(s, signPri)
		sign = append(s, sign...)
	}
	rst := wallet.Recover(address, sign, signData)
	if !rst {
		log.Printf("fail to recover, address: %x, sign:%x\n", address, sign)
		panic("fail to recover,block mine")
	}
	block.SetSign(sign)
	data := block.Output()

	b := DecodeBlock(data)
	if b.Key != block.Key {
		t.Errorf("error key:%x,%x", b.Key[:], block.Key[:])
	}
}
