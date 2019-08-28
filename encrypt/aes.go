package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
)

// AesEncrypt 加密的定义结构
type AesEncrypt struct {
	Key string
}

func (a *AesEncrypt) getKey() []byte {
	strKey := []byte(a.Key)
	var h = sha256.New()
	h.Write(strKey)
	h.Write(strKey)
	return h.Sum(nil)
}

// Encrypt 加密字符串
func (a *AesEncrypt) Encrypt(strMesg []byte) ([]byte, error) {
	key := a.getKey()
	var iv = []byte(key)[:aes.BlockSize]
	encrypted := make([]byte, len(strMesg))
	aesBlockEncrypter, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(encrypted, strMesg)
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
	return decrypted, nil
}
