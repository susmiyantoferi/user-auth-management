package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                 uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name               string          `gorm:"size:100;notnull" json:"name"`
	Email              string          `gorm:"uniqueIndex:user_deleted_at,where:deleted_at IS NULL;size:55;notnull" json:"email"`
	Password           string          `gorm:"size:100;notnull" json:"password"`
	Address            string          `gorm:"notnull" json:"address"`
	IsActive           bool            `gorm:"notnull;default:false" json:"is_active"`
	Role               RoleType        `gorm:"size:20" json:"role"`
	StatusVerification VerifType       `gorm:"size:20" json:"status_verification"`
	CreatedAt          time.Time       `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          *time.Time      `gorm:"type:timestamptz;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt          *gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type RoleType string
type VerifType string

const (
	RoleAdmin RoleType = "admin"
	RoleUser  RoleType = "user"

	Pending  = "pending"
	Verified = "verified"
)
