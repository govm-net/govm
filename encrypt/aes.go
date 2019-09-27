package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
)

// AesEncrypt 加密的定义结构
type AesEncrypt struct {
	Key string
}

const prefix = "encrypt"

func (a *AesEncrypt) getKey() []byte {
	strKey := []byte(a.Key)
	var h = sha256.New()
	h.Write([]byte(prefix))
	h.Write(strKey)
	return h.Sum(nil)
}

// Encrypt 加密字符串
func (a *AesEncrypt) Encrypt(strMesg []byte) ([]byte, error) {
	key := a.getKey()
	var iv = []byte(key)[:aes.BlockSize]
	msg := []byte("prefix")
	msg = append(msg, strMesg...)
	encrypted := make([]byte, len(msg))
	aesBlockEncrypter, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(encrypted, msg)
	return encrypted, nil
}

// Decrypt 解密字符串
func (a *AesEncrypt) Decrypt(src []byte) (strDesc []byte, err error) {
	defer func() {
		//错误处理
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	key := a.getKey()
	var iv = []byte(key)[:aes.BlockSize]
	decrypted := make([]byte, len(src))
	var aesBlockDecrypter cipher.Block
	aesBlockDecrypter, err = aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(decrypted, src)
	pre := []byte("prefix")
	if bytes.Compare(pre, decrypted[:len(pre)]) != 0 {
		return nil, errors.New("error password")
	}
	return decrypted[len(pre):], nil
}
