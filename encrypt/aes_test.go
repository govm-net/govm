package encrypt

import (
	"fmt"
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
	fmt.Println(string(strMsg))

	// Output: abcdef
}

func ExampleAesEncrypt_Decrypt() {
	aesEnc := AesEncrypt{"1234334"}
	arrEncrypt, err := aesEnc.Encrypt([]byte("abcdef"))
	if err != nil {
		log.Println(arrEncrypt)
		return
	}
	aesDec := AesEncrypt{"1234335"}
	strMsg, err := aesDec.Decrypt(arrEncrypt)
	if err != nil {
		fmt.Printf("error password")
		return
	}
	fmt.Println(string(strMsg))

	// Output: error password
}
