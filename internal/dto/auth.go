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
	OTP   string `json:"otp" validate:"required,max=6"`
}

type ResendOTPReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}
