package repository

import (
	"user-auth-exam/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Repository[entity.User]
	Deactivate(db *gorm.DB, id uuid.UUID, isActive bool) error
	FindByEmail(db *gorm.DB, email string) (*entity.User, error)
}

type userRepositoryImpl struct {
	repositoryImpl[entity.User]
}

func NewUserRepositoryImpl() UserRepository {
	return &userRepositoryImpl{}
}

func (u *userRepositoryImpl) Deactivate(db *gorm.DB, id uuid.UUID, isActive bool) error {
	return db.Model(&entity.User{}).Where("id = ?", id).Update("is_active", isActive).Error
}

func (u *userRepositoryImpl) FindByEmail(db *gorm.DB, email string) (*entity.User, error) {
	var user entity.User
	return &user, db.Where("email = ?", email).Take(&user).Error
}
