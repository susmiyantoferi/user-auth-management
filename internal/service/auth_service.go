package service

import (
	"context"
	"crypto/hmac"
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
	"gorm.io/gorm/clause"
)

type AuthService interface {
	Login(c context.Context, req *dto.LoginUserRequest) (*dto.TokenResponse, error)
	Register(c context.Context, req *dto.RegisterUserRequest) error
	Logout(c context.Context, userID uuid.UUID) error
	Refresh(c context.Context, req *dto.ResfreshTokenRequest) (*dto.TokenResponse, error)
	VerifyOTP(c context.Context, req *dto.VerifyOTPReq) error
	ResendOTP(c context.Context, req *dto.ResendOTPReq) error
	ForgotPassword(c context.Context, req *dto.ForgotPasswordReq) error
	ResetPassword(c context.Context, req *dto.ResetPasswordRequest) error
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

const (
	OtpExpiretion     = 5 * time.Minute
	maxAttemp         = 5
	TimeCooldown      = 60 * time.Second
	PwResetExpiretion = 10 * time.Minute
)

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

	email := strings.ToLower(req.Email)
	pass, err := encrypt.BcryptText(req.Password)
	if err != nil {
		s.Log.WithError(err).Error("failed to bcrypt password")
		return ErrInternalServer
	}

	if err := s.Db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		user := entity.User{
			Name:               req.Name,
			Email:              email,
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

	otpHash := hashing.OtpToHash(otp, s.Key.HmacSecret)

	//set key redis
	keyRdsOtp := fmt.Sprintf("otp:email:%s", email)

	//save to redis
	if err := s.Redis.Set(c, keyRdsOtp, hex.EncodeToString(otpHash), OtpExpiretion).Err(); err != nil {
		s.Log.WithError(err).Error("register: failed save otp redis")
		return ErrInternalServer
	}

	//send email for otp
	if err := s.EmailService.SendOTP(c, email, otp); err != nil {
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

func (s *authServiceImpl) VerifyOTP(c context.Context, req *dto.VerifyOTPReq) error {
	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	email := strings.ToLower(req.Email)
	keyOtp := fmt.Sprintf("otp:email:%s", email)
	attempKey := fmt.Sprintf("otp:attemp:email:%s", email)

	otpHasHex, err := s.Redis.Get(c, keyOtp).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.Log.WithError(err).Error("otp not found in redis")
			return ErrOtpInvalid
		}
		s.Log.WithError(err).Error("verify: failed get otp redis")
		return ErrInternalServer
	}

	//sett rate to hit verify otp
	attemp, err := s.Redis.Incr(c, attempKey).Result()
	if err != nil {
		return ErrInternalServer
	}

	if attemp == 1 {
		if err := s.Redis.Expire(c, attempKey, OtpExpiretion).Err(); err != nil {
			return ErrInternalServer
		}
	}

	if attemp > maxAttemp {
		return ErrToManyReq
	}

	inputOtp := hashing.OtpToHash(req.OTP, s.Key.HmacSecret)

	storeOtpHas, err := hex.DecodeString(otpHasHex)
	if err != nil {
		return ErrInternalServer
	}

	//compare apakah sama
	if !hmac.Equal(inputOtp, storeOtpHas) {
		return ErrOtpInvalid
	}

	//get user & update
	if err := s.Db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var user entity.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email = ?", email).Take(&user).Error; err != nil {
			s.Log.WithError(err).Error("verify: failed get user by email")
			return err
		}

		updateUser := entity.User{
			IsActive:           true,
			StatusVerification: entity.Verified,
		}

		if err := tx.Model(&user).Updates(updateUser).Error; err != nil {
			s.Log.WithError(err).Error("verify: failed update user status")
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if err := s.Redis.Del(c, keyOtp).Err(); err != nil {
		s.Log.WithError(err).Error("verify: failed delete otp redis")
		return ErrInternalServer
	}

	return nil

}

func (s *authServiceImpl) ResendOTP(c context.Context, req *dto.ResendOTPReq) error {
	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	email := strings.ToLower(req.Email)

	user, err := s.UserRepo.FindByEmail(s.Db.WithContext(c), email)
	if err != nil {
		s.Log.WithError(err).Error("resend otp: failed get user")
		return ErrInternalServer
	}

	//validate status if verify
	if user.StatusVerification == entity.Verified {
		return ErrUserIsVerified
	}

	cooldownKey := fmt.Sprintf("otp:resend:%s", user.Email)

	exists, err := s.Redis.Exists(c, cooldownKey).Result()
	if err != nil {
		s.Log.WithError(err).Error("resend otp: failed check cooldown")
		return ErrInternalServer
	}

	if exists > 0 {
		return ErrCooldown
	}

	newOtp, err := otp.GenerateOTP()
	if err != nil {
		s.Log.WithError(err).Error("resend otp: failed generate otp")
		return ErrInternalServer
	}

	otpHash := hashing.OtpToHash(newOtp, s.Key.HmacSecret)

	otpKey := fmt.Sprintf("otp:email:%s", user.Email)

	//set new otp
	if err := s.Redis.Set(c, otpKey, hex.EncodeToString(otpHash), OtpExpiretion).Err(); err != nil {
		s.Log.WithError(err).Error("resend otp: failed save otp redis")
		return ErrInternalServer
	}

	//set cooldown for resend otp
	acquired, err := s.Redis.SetNX(c, cooldownKey, "1", TimeCooldown).Result()
	if err != nil {
		s.Log.WithError(err).Error("resend otp: failed set cooldown redis")
		return ErrInternalServer
	}

	if !acquired {
		s.Log.Warn("resend otp: cooldown key already exists")
		return ErrCooldown
	}

	//send new otp to email
	if err := s.EmailService.SendOTP(c, user.Email, newOtp); err != nil {
		s.Log.WithError(err).Error("resend otp: failed send email")
		return ErrInternalServer
	}

	return nil
}

func (s *authServiceImpl) ForgotPassword(c context.Context, req *dto.ForgotPasswordReq) error {
	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	email := strings.ToLower(req.Email)

	user, err := s.UserRepo.FindByEmail(s.Db.WithContext(c), email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Log.WithError(err).Error("forgot password: user not found")
			return nil
		}
		s.Log.WithError(err).Error("forgot password: failed get user by email")
		return ErrInternalServer
	}

	//set cooldown
	cooldownKey := fmt.Sprintf("reset-password:cooldown:%s", user.Email)
	acquired, err := s.Redis.SetNX(c, cooldownKey, "1", TimeCooldown).Result()
	if err != nil {
		s.Log.WithError(err).Error("forgot password: failed set cooldown")
		return ErrInternalServer
	}

	if !acquired {
		s.Log.Warn("forgot password: cooldown key already exists")
		return ErrCooldown
	}

	token, err := token.GenerateResetToken()
	if err != nil {
		s.Log.WithError(err).Error("forgot password: failed generate token")
		return ErrInternalServer
	}

	tokenHash := hashing.HashResetToken(token)

	//set tokwn to redis
	resetPwKey := fmt.Sprintf("reset-password:%s", tokenHash)
	if err := s.Redis.Set(c, resetPwKey, user.ID.String(), PwResetExpiretion).Err(); err != nil {
		s.Log.WithError(err).Error("forgot password: failed save reset token redis")
		return ErrInternalServer
	}

	//send email reset pw
	if err := s.EmailService.SendPasswordReset(c, user.Email, token); err != nil {
		s.Log.WithError(err).Error("forgot password: failed send reset email")
		return ErrInternalServer
	}

	return nil

}

func (s *authServiceImpl) ResetPassword(c context.Context, req *dto.ResetPasswordRequest) error {
	if err := s.Validate.Struct(req); err != nil {
		s.Log.WithError(err).Error("Validation failed")
		return ErrValidationFailed
	}

	tokenHash := hashing.HashResetToken(req.Token)

	resetKey := fmt.Sprintf("reset-password:%s", tokenHash)
	id, err := s.Redis.Get(c, resetKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.Log.WithError(err).Error("reset password: token not found in redis")
			return ErrTokenInvalid
		}
		s.Log.WithError(err).Error("reset password: failed get token redis")
		return ErrInternalServer
	}

	if err := s.Db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var user entity.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				s.Log.WithError(err).Error("reset password: not found user")
				return err
			}
			s.Log.WithError(err).Error("reset password: failed get user by id")
			return err
		}

		pass, err := encrypt.BcryptText(req.Password)
		if err != nil {
			s.Log.WithError(err).Error("reset password: failed bcrypt password")
			return ErrInternalServer
		}

		u := entity.User{
			Password: pass,
		}

		if err := tx.Model(&user).Updates(u).Error; err != nil {
			s.Log.WithError(err).Error("reset password: failed update password")
			return err
		}

		return nil
	}); err != nil {
		return ErrInternalServer
	}

	//del token reset pw
	if err := s.Redis.Del(c, resetKey).Err(); err != nil {
		s.Log.WithError(err).Error("reset password: failed delete resetKey redis")
		return ErrInternalServer
	}

	//del cred login
	accessKey := fmt.Sprintf("jwt:%s", id)
	refreshKey := fmt.Sprintf("refresh:%s", id)
	if err := s.Redis.Del(c, accessKey, refreshKey).Err(); err != nil {
		s.Log.WithError(err).Error("reset password: failed delete accessKey or refreshKey redis")
		return ErrInternalServer
	}

	return nil

}
