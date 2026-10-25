package hashing

import (
	"crypto/hmac"
	"crypto/sha256"
)

func StrToHash(s string, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(s))
	return mac.Sum(nil)
}
