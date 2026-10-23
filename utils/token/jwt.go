package token

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/dto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateToken(cfg *config.JwtConfig, userId uuid.UUID, email, role string, exp time.Duration) (string, error) {
	jwtSecret := []byte(cfg.SecretKey)
	jwtExp := time.Now().Add(exp * time.Hour)

	tokenCLaim := &dto.TokenClaim{
		UserID: userId,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(jwtExp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenCLaim)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenStr, nil
}

func GenerateRefreshToken(userID uuid.UUID, cfg *config.JwtConfig) (string, time.Time, error) {
	expiry := time.Now().Add(time.Duration(cfg.RefreshExpiredTime) * time.Hour)
	rawToken := fmt.Sprintf("%s:%d:%s", userID.String(), expiry.Unix(), time.Now().String())

	h := sha256.New()
	h.Write([]byte(rawToken))
	hash := hex.EncodeToString(h.Sum(nil))

	return hash, expiry, nil
}

func ClaimTokenRefresh(tokenUser string, cfg *config.JwtConfig) (*dto.TokenClaim, error) {
	jwtSecret := []byte(cfg.SecretKey)
	claim := &dto.TokenClaim{}

	token, err := jwt.ParseWithClaims(tokenUser, claim, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")

	}

	return claim, nil
}
