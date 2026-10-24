package dto

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenClaim struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

type ResfreshTokenRequest struct {
	TokenRefresh string `json:"token_refresh" validate:"required"`
}

type TokenResponse struct {
	Token        string    `json:"access_token"`
	RefreshToken *string   `json:"refresh_token,omitempty"`
	ExpiresIn    time.Time `json:"expires_in"`
	CreatedAt    time.Time `json:"created_at"`
}
