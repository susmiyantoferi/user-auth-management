package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/entity"
	"user-auth-exam/internal/repository"
	"user-auth-exam/utils/encrypt"
	"user-auth-exam/utils/token"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AuthService interface {
	Login(c context.Context, req *dto.LoginUserRequest) (*dto.TokenResponse, error)
	Register(c context.Context, req *dto.RegisterUserRequest) error
	Logout(c context.Context, userID uuid.UUID) error
}

type authServiceImpl struct {
	UserRepo repository.UserRepository
	Db       *gorm.DB
	Validate *validator.Validate
	Log      *logrus.Logger
	Jwt      *config.JwtConfig
	Redis    *redis.Client
}

func NewAuthServiceImpl(userRepo repository.UserRepository, db *gorm.DB, validate *validator.Validate, log *logrus.Logger, jwt *config.JwtConfig, redis *redis.Client) AuthService {
	return &authServiceImpl{
		UserRepo: userRepo,
		Db:       db,
		Validate: validate,
		Log:      log,
		Jwt:      jwt,
		Redis:    redis,
	}
}

func (s *authServiceImpl) Login(c context.Context, req *dto.LoginUserRequest) (*dto.TokenResponse, error) {

	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return nil, ErrValidationFailed
	}

	user, err := s.UserRepo.FindByEmail(s.Db.WithContext(c), req.Email)
	if err != nil {
		s.Log.WithError(err).Error("failed get user by email")
		return nil, ErrCredentialInvalid
	}

	if !user.IsActive || user.StatusVerification == entity.Pending {
		return nil, ErrUserInvalidStatus
	}

	if !encrypt.CompareBcryptText(user.Password, req.Password) {
		return nil, ErrCredentialInvalid
	}

	accessToken, err := token.GenerateToken(s.Jwt, user.ID, user.Email, string(user.Role), time.Duration(s.Jwt.ExpiredTime))
	if err != nil {
		s.Log.WithError(err).Error("failed generate token")
		return nil, ErrInternalServer
	}

	refreshToken, _, err := token.GenerateRefreshToken(user.ID, s.Jwt)
	if err != nil {
		s.Log.WithError(err).Error("failed generate refresh token")
		return nil, ErrInternalServer
	}

	//set token in redis with ttl
	accessTTL := time.Duration(s.Jwt.ExpiredTime) * time.Hour
	refreshTTL := time.Duration(s.Jwt.RefreshExpiredTime) * time.Hour
	timeNow := time.Now()

	if err := s.Redis.Set(c, "jwt:"+user.ID.String(), accessToken, accessTTL).Err(); err != nil {
		s.Log.Errorf("failed to save access token to redis: %v", err)
		return nil, ErrInternalServer
	}
	if err := s.Redis.Set(c, "refresh:"+user.ID.String(), refreshToken, refreshTTL).Err(); err != nil {
		s.Log.Errorf("failed to save refresh token to redis: %v", err)
		return nil, ErrInternalServer
	}

	tokenCreated := dto.TokenResponse{
		Token:        accessToken,
		TokenRefresh: refreshToken,
		ExpiresIn:    timeNow.Add(accessTTL),
		CreatedAt:    timeNow,
	}

	return &tokenCreated, nil
}

func (s *authServiceImpl) Register(c context.Context, req *dto.RegisterUserRequest) error {

	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	pass, err := encrypt.BcryptText(req.Password)
	if err != nil {
		s.Log.WithError(err).Error("failed to bcrypt password")
		return ErrInternalServer
	}

	user := entity.User{
		Name:               req.Name,
		Email:              req.Email,
		Password:           pass,
		Address:            req.Address,
		Role:               entity.RoleUser,
		StatusVerification: entity.Pending,
	}

	if err := s.UserRepo.Create(s.Db.WithContext(c), &user); err != nil {
		s.Log.WithError(err).Error("Failed to create user")
		return ErrInternalServer
	}

	return nil
}

func (s *authServiceImpl) Logout(c context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		s.Log.Infof("logout: userID is nil")
		return ErrValidationFailed
	}

	user, err := s.UserRepo.GetByID(s.Db.WithContext(c), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("logout: user not found")
			return ErrRecordNotFound
		}

		s.Log.WithError(err).Error("logout: failed get user by id")
		return ErrInternalServer
	}

	strID := user.ID.String()

	keyJwt := fmt.Sprintf("jwt:%s", strID)
	keyRefresh := fmt.Sprintf("refresh:%s", strID)

	if err := s.Redis.Del(c, keyJwt).Err(); err != nil {
		s.Log.WithError(err).Error("logout: failed delete jwt token")
		return ErrInternalServer
	}

	if err := s.Redis.Del(c, keyRefresh).Err(); err != nil {
		s.Log.WithError(err).Error("logout: failed delete refresh token")
		return ErrInternalServer
	}

	return nil
}
