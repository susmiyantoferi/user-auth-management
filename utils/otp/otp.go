package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateOTP() (string, error) {
	n, err := rand.Int(
		rand.Reader,
		big.NewInt(1000000),
	)
	if err != nil {
		return "", err
	}
	strOtp := fmt.Sprintf("%06d", n.Int64())
	return strOtp, nil
}
