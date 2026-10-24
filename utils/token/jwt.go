package token

import (
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
	now := time.Now()

	tokenCLaim := &dto.TokenClaim{
		UserID:    userId,
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(jwtExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenCLaim)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenStr, nil
}

func GenerateRefreshToken(userID uuid.UUID, cfg *config.JwtConfig) (string, error) {
	now := time.Now()
	jwtSecret := []byte(cfg.SecretKey)
	expiry := now.Add(
		time.Duration(cfg.RefreshExpiredTime) * time.Hour,
	)

	claims := &dto.TokenClaim{
		UserID:    userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenStr, nil

}

func ClaimTokenRefresh(tokenString string, cfg *config.JwtConfig) (*dto.TokenClaim, error) {

	claims := &dto.TokenClaim{}

	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %v",
					t.Header["alg"],
				)
			}

			return []byte(cfg.SecretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.TokenType != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}

	return claims, nil
}
