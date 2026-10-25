package dto

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Address  string `json:"address" validate:"required"`
}

type VerifyOTPReq struct {
	Email string `json:"email" validate:"required,email"`
	OTP string `json:"otp" validate:"required,max=6"`
}