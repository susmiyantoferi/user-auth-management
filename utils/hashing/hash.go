package hashing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func OtpToHash(s string, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(s))
	return mac.Sum(nil)
}

func HashResetToken(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
