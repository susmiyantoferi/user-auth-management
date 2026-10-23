package dto

import (
	"time"

	"github.com/google/uuid"
)

type UpdateEmailRequest struct {
	ID    uuid.UUID `json:"id" validate:"required"`
	Email string    `json:"email" validate:"required,email"`
}

type UpdateUserRequest struct {
	ID      uuid.UUID `json:"id" validate:"required"`
	Name    string    `json:"name" validate:"required"`
	Address string    `json:"address" validate:"required"`
}

type UserResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Address            string    `json:"address"`
	IsActive           *bool     `json:"is_active,omitempty"`
	Role               string    `json:"role"`
	StatusVerification string    `json:"status_verification"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
