package tool

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/sha256"

	"github.com/doraemonkeys/doraemon"
)

type AES192Key []byte

func HashToAES192Key(c []byte) AES192Key {
	// if len(c) == 0 {
	// 	panic("unreachable: Invalid input string")
	// }
	hash := doraemon.ComputeSHA256(bytes.NewReader(c)).Unwrap()
	return hash[:192/8]
}

func HashToAES192Key2(c []byte) AES192Key {
	iters := 300000
	hash := c
	for range iters {
		hash = doraemon.ComputeSHA256(bytes.NewReader(hash)).Unwrap()
	}
	return hash[:192/8]
}

func AES192KeyKDF(password string, salt []byte) AES192Key {
	iterations := 10000
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, 192/8)
	if err != nil {
		panic("unreachable: " + err.Error())
	}
	return key
}

func AES192KeyKDF2(password string, salt []byte) AES192Key {
	iterations := 200000
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, 192/8)
	if err != nil {
		panic("unreachable: " + err.Error())
	}
	return key
}
