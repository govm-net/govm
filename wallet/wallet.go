package wallet

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	// "encoding/hex"
	"github.com/btcsuite/btcd/btcec"
	"log"
	"time"
)

const (
	// EAddrTypeDefault the type of default public address
	EAddrTypeDefault = byte(iota + 1)
	// EAddrTypeIBS identity-based signature 基于身份的签名，不同时间，使用不同私钥签名(签名时间是消息的前8个字节)
	EAddrTypeIBS
)

const (
	// AddressLength address length
	AddressLength = 24
	// SignLen default length of sign
	SignLen       = 65
	publicKeyLen  = 33
	privateKeyLen = 32
	// TimeDuration EAddrTypeIBS的子私钥有效时间,一个月
	TimeDuration = 31558150000 / 12
)

func init() {
	var data = make([]byte, 10)
	rand.Read(data)
}

// NewPrivateKey 获取一个随机的私钥
func NewPrivateKey() []byte {
	priKey, _ := btcec.NewPrivateKey(btcec.S256())
	out := priKey.Serialize()
	out = append(out, []byte(time.Now().String())...)
	return getHash(out)
}

// NewChildPrivateKeyOfIBS create child key of the address,time(ms)
func NewChildPrivateKeyOfIBS(privK []byte, t uint64) (cPriKey []byte, signPre []byte) {
	address := PublicKeyToAddress(GetPublicKey(privK), EAddrTypeIBS)
	cPriKey = NewPrivateKey()
	cPub := GetPublicKey(cPriKey)

	msgT := t + bytesToUint64(address)
	msgT /= TimeDuration

	buf1 := new(bytes.Buffer)
	binary.Write(buf1, binary.BigEndian, msgT)
	binary.Write(buf1, binary.BigEndian, cPub)

	signPre = Sign(privK, buf1.Bytes())
	return
}

// GetDeadlineOfIBS get deadline of child key by public address
func GetDeadlineOfIBS(addr []byte) uint64 {
	t1 := bytesToUint64(addr)
	msgT := uint64(time.Now().Unix()*1000) + t1
	msgT /= TimeDuration
	msgT = (msgT+1)*TimeDuration - t1
	return msgT
}

// GetPublicKey 通过私钥获得公钥
func GetPublicKey(in []byte) []byte {
	out := []byte{}
	for i := 0; i < len(in)/privateKeyLen; i++ {
		st := i * privateKeyLen
		end := (i + 1) * privateKeyLen
		_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), in[st:end])
		out = append(out, pubKey.SerializeCompressed()...)
	}

	return out
}

// PublicKeyToAddress 将公钥转成钱包地址
func PublicKeyToAddress(in []byte, addrType uint8) []byte {
	var out [AddressLength]byte
	switch addrType {
	case EAddrTypeDefault:
		if len(in) != publicKeyLen {
			log.Println("error public key length")
			return nil
		}
	case EAddrTypeIBS:
		if len(in) != publicKeyLen {
			log.Println("error public key length")
			return nil
		}
	default:
		log.Println("unsupport:", addrType)
		return nil
	}

	in = append(in, addrType)
	h := getHash(in)
	buf := bytes.NewReader(h)
	binary.Read(buf, binary.BigEndian, &out)
	out[0] = addrType

	return out[:]
}

func getHash(msg []byte) []byte {
	sha := sha256.New()
	sha.Write(msg)
	return sha.Sum(nil)
}

// Sign 用私钥对msg进行签名
func Sign(privK, msg []byte) []byte {
	msgH := getHash(msg)
	if len(privK) != privateKeyLen {
		return nil
	}
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privK)
	signature, err := btcec.SignCompact(btcec.S256(), privKey, msgH, true)
	if err != nil {
		log.Println(err)
		return nil
	}
	//log.Printf("sign length:%d,hash:%x\n", len(msg), msgH)

	return signature
}

func getPublicByRecover(sign, msgH []byte) []byte {
	pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sign, msgH)
	if err != nil {
		log.Println(err)
		return nil
	}
	if !wasCompressed {
		log.Println(wasCompressed)
		return nil
	}
	return pk.SerializeCompressed()
}

func bytesToUint64(data []byte) uint64 {
	if len(data) < 8 {
		return 0
	}
	var t uint64
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.BigEndian, &t)
	return t
}

// Recover 通过签名信息，提取钱包地址
func Recover(address, sign, msg []byte) bool {
	msgH := getHash(msg)
	//log.Printf("recover length:%d,hash:%x\n", len(msg), msgH)
	publicKey := []byte{}
	switch address[0] {
	case EAddrTypeDefault:
		if len(sign) != SignLen {
			return false
		}
		publicKey = getPublicByRecover(sign, msgH)
	case EAddrTypeIBS:
		if len(sign)%SignLen != 0 {
			return false
		}
		if len(sign)/SignLen != 2 {
			return false
		}
		if len(msg) <= 8 {
			return false
		}
		s1 := sign[0:SignLen]
		s2 := sign[SignLen:]
		// log.Println("Recover signPri:", hex.EncodeToString(s1))
		// log.Println("Recover signSuf:", hex.EncodeToString(s2))
		pk := getPublicByRecover(s2, msgH)
		// log.Println("Recover childPubK:", hex.EncodeToString(pk))

		//msg1 := pk
		msgT := bytesToUint64(msg) + bytesToUint64(address[:])
		msgT /= TimeDuration

		buf1 := new(bytes.Buffer)
		binary.Write(buf1, binary.BigEndian, msgT)
		binary.Write(buf1, binary.BigEndian, pk)

		//msg1 = append(msg1, buf1.Bytes()...)
		//log.Println("Recover msg1:", hex.EncodeToString(msg1))

		msgH = getHash(buf1.Bytes())
		publicKey = getPublicByRecover(s1, msgH)
		// log.Println("Recover parentPubK:", hex.EncodeToString(publicKey))
	}

	pkAddr := PublicKeyToAddress(publicKey, address[0])
	if bytes.Compare(pkAddr, address) != 0 {
		return false
	}

	return true
}
