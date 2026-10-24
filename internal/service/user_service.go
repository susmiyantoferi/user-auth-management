package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/entity"
	"user-auth-exam/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserService interface {
	Update(c context.Context, req *dto.UpdateUserRequest) error
	Deactivate(c context.Context, id uuid.UUID) error
	GetByID(c context.Context, id uuid.UUID) (*dto.UserResponse, error)
	ChangeEmail(c context.Context, req *dto.UpdateEmailRequest) error
}

type userServiceImpl struct {
	UserRepo    repository.UserRepository
	Db          *gorm.DB
	Validate    *validator.Validate
	Log         *logrus.Logger
	RedisClient *redis.Client
}

func NewUserServiceImpl(userRepo repository.UserRepository, db *gorm.DB, validate *validator.Validate, log *logrus.Logger, redisClient *redis.Client) UserService {
	return &userServiceImpl{
		UserRepo:    userRepo,
		Db:          db,
		Validate:    validate,
		Log:         log,
		RedisClient: redisClient,
	}
}

var (
	ErrValidationFailed  = errors.New("Validation failed")
	ErrRecordNotFound    = errors.New("Record not found")
	ErrCredentialInvalid = errors.New("Invalid email or password")
	ErrUserInvalidStatus = errors.New("User will be verified or active")
	ErrInternalServer    = errors.New("Internal server error")
	ErrEmailExist        = errors.New("Email already exits")
)

func (s *userServiceImpl) Update(c context.Context, req *dto.UpdateUserRequest) error {
	db := s.Db.WithContext(c)

	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	user, err := s.UserRepo.GetByID(db, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("get user by id not found")
			return ErrRecordNotFound
		}
		s.Log.WithError(err).Error("failed get user by id")
		return ErrInternalServer
	}

	if !user.IsActive || user.StatusVerification == entity.Pending {
		s.Log.Infof("user must be active and verified")
		return ErrUserInvalidStatus
	}

	u := &entity.User{
		Name:    req.Name,
		Address: req.Address,
	}

	if err := s.UserRepo.Update(db, u, user.ID); err != nil {
		s.Log.WithError(err).Error("failed to update user")
		return ErrInternalServer
	}

	return nil
}

func (s *userServiceImpl) Deactivate(c context.Context, id uuid.UUID) error {
	db := s.Db.WithContext(c)

	user, err := s.UserRepo.GetByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("get user by id not found")
			return ErrRecordNotFound
		}
		s.Log.WithError(err).Error("failed get user by id")
		return ErrInternalServer
	}

	keyJwt := fmt.Sprintf("jwt:%d", user.ID)
	keyRefresh := fmt.Sprintf("refresh:%d", user.ID)

	if user.IsActive {
		if err := s.RedisClient.Del(c, keyJwt).Err(); err != nil {
			s.Log.WithError(err).Error("failed delete token redis")
			return ErrInternalServer
		}

		if err := s.RedisClient.Del(c, keyRefresh).Err(); err != nil {
			s.Log.WithError(err).Error("failed delete token refresh redis")
			return ErrInternalServer
		}
	}

	user.IsActive = !user.IsActive

	if err := s.UserRepo.Deactivate(db, user.ID, user.IsActive); err != nil {
		s.Log.WithError(err).Error("failed deactive user")
		return ErrInternalServer
	}

	return nil
}

func (s *userServiceImpl) GetByID(c context.Context, id uuid.UUID) (*dto.UserResponse, error) {
	db := s.Db.WithContext(c)

	user, err := s.UserRepo.GetByID(db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("get user by id not found")
			return nil, ErrRecordNotFound
		}
		s.Log.WithError(err).Error("failed get user by id")
		return nil, ErrInternalServer
	}

	return &dto.UserResponse{
		ID:                 user.ID.String(),
		Name:               user.Name,
		Email:              user.Email,
		Address:            user.Address,
		IsActive:           &user.IsActive,
		Role:               string(user.Role),
		StatusVerification: string(user.StatusVerification),
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          *user.UpdatedAt,
	}, nil

}

func (s *userServiceImpl) ChangeEmail(c context.Context, req *dto.UpdateEmailRequest) error {
	db := s.Db.WithContext(c)

	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	user, err := s.UserRepo.GetByID(db, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("get user by id not found")
			return ErrRecordNotFound
		}
		s.Log.WithError(err).Error("user: failed get user")
		return ErrInternalServer
	}

	if !user.IsActive || user.StatusVerification == entity.Pending {
		s.Log.Infof("user must be active and verified")
		return ErrUserInvalidStatus
	}

	update := &entity.User{
		Email: req.Email,
	}

	if err := s.UserRepo.Update(db, update, user.ID); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			s.Log.WithError(err).Error("user: email already exist")
			return ErrEmailExist
		}
		s.Log.WithError(err).Error("user: failed change email")
		return ErrInternalServer
	}

	return nil
}
