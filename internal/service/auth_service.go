package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/entity"
	"user-auth-exam/internal/repository"
	"user-auth-exam/utils/encrypt"
	"user-auth-exam/utils/hashing"
	"user-auth-exam/utils/otp"
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
	Refresh(c context.Context, req *dto.ResfreshTokenRequest) (*dto.TokenResponse, error)
}

type authServiceImpl struct {
	UserRepo     repository.UserRepository
	Db           *gorm.DB
	Validate     *validator.Validate
	Log          *logrus.Logger
	Jwt          *config.JwtConfig
	Redis        *redis.Client
	Key          *config.SecretKey
	EmailService EmailService
}

func NewAuthServiceImpl(userRepo repository.UserRepository, db *gorm.DB, validate *validator.Validate, log *logrus.Logger, jwt *config.JwtConfig, redis *redis.Client, key *config.SecretKey, emailService EmailService) AuthService {
	return &authServiceImpl{
		UserRepo:     userRepo,
		Db:           db,
		Validate:     validate,
		Log:          log,
		Jwt:          jwt,
		Redis:        redis,
		Key:          key,
		EmailService: emailService,
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

	refreshToken, err := token.GenerateRefreshToken(user.ID, s.Jwt)
	if err != nil {
		s.Log.WithError(err).Error("failed generate refresh token")
		return nil, ErrInternalServer
	}

	//set token in redis with ttl
	accessTTL := time.Duration(s.Jwt.ExpiredTime) * time.Hour
	refreshTTL := time.Duration(s.Jwt.RefreshExpiredTime) * time.Hour
	timeNow := time.Now()

	if err := s.Redis.Set(c, "jwt:"+user.ID.String(), accessToken, accessTTL).Err(); err != nil {
		s.Log.WithError(err).Error("failed to save access token to redis")
		return nil, ErrInternalServer
	}
	if err := s.Redis.Set(c, "refresh:"+user.ID.String(), refreshToken, refreshTTL).Err(); err != nil {
		s.Log.WithError(err).Error("failed to save refresh token to redis")
		return nil, ErrInternalServer
	}

	tokenCreated := dto.TokenResponse{
		Token:        accessToken,
		RefreshToken: &refreshToken,
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

	if err := s.Db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		user := entity.User{
			Name:               req.Name,
			Email:              req.Email,
			Password:           pass,
			Address:            req.Address,
			Role:               entity.RoleUser,
			StatusVerification: entity.Pending,
		}

		if err := s.UserRepo.Create(tx, &user); err != nil {
			if strings.Contains(err.Error(), "duplicate key value") {
				s.Log.WithError(err).Error("user: email already exist")
				return ErrEmailExist
			}
			s.Log.WithError(err).Error("Failed to create user")
			return ErrInternalServer
		}
		return nil
	}); err != nil {
		s.Log.WithError(err).Error("Failed to register user")
		return ErrInternalServer
	}

	otp, err := otp.GenerateOTP()
	if err != nil {
		s.Log.WithError(err).Error("Failed generate otp")
		return ErrInternalServer
	}

	otpHash := hashing.StrToHash(otp, s.Key.HmacSecret)

	//set email to lower case
	email := strings.ToLower(req.Email)

	//set key redis
	keyRdsOtp := fmt.Sprintf("otp:email:%s", email)
	ttl := 5 * time.Minute

	//save to redis
	if err := s.Redis.Set(c, keyRdsOtp, hex.EncodeToString(otpHash), ttl).Err(); err != nil {
		s.Log.WithError(err).Error("register: failed save otp redis")
		return ErrInternalServer
	}

	//send email for otp
	if err := s.EmailService.SendEmail(c, email, otp); err != nil {
		s.Log.WithError(err).Error("register: failed send email")
		return ErrInternalServer
	}

	return nil
}

func (s *authServiceImpl) Logout(c context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		s.Log.Infof("logout: userID is nil")
		return ErrValidationFailed
	}

	strID := userID.String()

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

func (s *authServiceImpl) Refresh(c context.Context, req *dto.ResfreshTokenRequest) (*dto.TokenResponse, error) {
	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return nil, ErrValidationFailed
	}

	claimUser, err := token.ClaimTokenRefresh(req.TokenRefresh, s.Jwt)
	if err != nil {
		s.Log.WithError(err).Error("refresh: invalid refresh token")
		return nil, ErrTokenInvalid
	}

	keyRefresh := fmt.Sprintf("refresh:%s", claimUser.UserID.String())
	tokenRedis, err := s.Redis.Get(c, keyRefresh).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.Log.WithError(err).Error("refresh: invalid refresh token")
			return nil, ErrCredentialInvalid
		}
		s.Log.WithError(err).Error("refresh: failed get refresh token from redis")
		return nil, ErrInternalServer
	}

	//cekkk token
	if tokenRedis != req.TokenRefresh {
		s.Log.Warnf("refresh: token refresh invalid")
		return nil, ErrTokenInvalid
	}

	user, err := s.UserRepo.GetByID(s.Db.WithContext(c), claimUser.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("refresh: user not found")
			return nil, ErrRecordNotFound
		}

		s.Log.WithError(err).Error("refresh: failed get user by id")
		return nil, ErrInternalServer
	}

	//generate access token baru
	newToken, err := token.GenerateToken(s.Jwt, user.ID, user.Email, string(user.Role), time.Duration(s.Jwt.ExpiredTime))
	if err != nil {
		s.Log.WithError(err).Error("refresh: failed generate token")
		return nil, ErrInternalServer
	}

	accessTTL := time.Duration(s.Jwt.ExpiredTime) * time.Hour
	keyAccess := fmt.Sprintf("jwt:%s", user.ID.String())
	timeNow := time.Now()

	if err := s.Redis.Set(c, keyAccess, newToken, accessTTL).Err(); err != nil {
		s.Log.WithError(err).Error("failed to save access token to redis")
		return nil, ErrInternalServer
	}

	return &dto.TokenResponse{
		Token:        newToken,
		RefreshToken: nil,
		ExpiresIn:    timeNow.Add(accessTTL),
		CreatedAt:    timeNow,
	}, nil

}
