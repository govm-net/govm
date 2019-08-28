package encrypt

import (
	"log"
)

func ExampleAesEncrypt_Encrypt() {
	aesEnc := AesEncrypt{"1234334"}
	arrEncrypt, err := aesEnc.Encrypt([]byte("abcdef"))
	if err != nil {
		log.Println(arrEncrypt)
		return
	}
	strMsg, err := aesEnc.Decrypt(arrEncrypt)
	if err != nil {
		log.Println(arrEncrypt)
		return
	}
	log.Println(string(strMsg))

	// Output: abcdef
}
