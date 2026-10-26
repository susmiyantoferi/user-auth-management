package controller

import (
	"errors"
	"net/http"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/response"
	"user-auth-exam/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthController interface {
	Login(c *gin.Context)
	Register(c *gin.Context)
	Logout(c *gin.Context)
	Refresh(c *gin.Context)
	Verify(c *gin.Context)
	Resend(c *gin.Context)
	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
	ChangePassword(c *gin.Context)
}

type authControllerImpl struct {
	AuthService service.AuthService
}

func NewAuthControllerImpl(authService service.AuthService) AuthController {
	return &authControllerImpl{
		AuthService: authService,
	}
}

func (a *authControllerImpl) Login(c *gin.Context) {
	var body dto.LoginUserRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	resp, err := a.AuthService.Login(c.Request.Context(), &body)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCredentialInvalid):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))

		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))

		case errors.Is(err, service.ErrUserInvalidStatus):
			c.JSON(http.StatusForbidden, response.ErrorResponse(err.Error()))

		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}

		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(resp))
}

func (a *authControllerImpl) Register(c *gin.Context) {
	var req dto.RegisterUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	err := a.AuthService.Register(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrEmailExist):
			c.JSON(http.StatusConflict, response.ErrorResponse(err.Error()))

		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, response.SuccessResponse(nil, "success, check your email for verification code"))

}

func (a *authControllerImpl) Logout(c *gin.Context) {

	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)

	if err := a.AuthService.Logout(c.Request.Context(), user.UserID); err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrInternalServer):
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil))
}

func (a *authControllerImpl) Refresh(c *gin.Context) {
	var req dto.ResfreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	resp, err := a.AuthService.Refresh(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrTokenInvalid):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(resp))
}

func (a *authControllerImpl) Verify(c *gin.Context) {
	var body dto.VerifyOTPReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	if err := a.AuthService.VerifyOTP(c.Request.Context(), &body); err != nil {
		switch {
		case errors.Is(err, service.ErrOtpInvalid):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrToManyReq):
			c.JSON(http.StatusTooManyRequests, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil))

}

func (a *authControllerImpl) Resend(c *gin.Context) {
	var body dto.ResendOTPReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	if err := a.AuthService.ResendOTP(c.Request.Context(), &body); err != nil {
		switch {
		case errors.Is(err, service.ErrUserIsVerified):
			c.JSON(http.StatusForbidden, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrCooldown):
			c.JSON(http.StatusTooManyRequests, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success, check your email for verification code"))

}

func (a authControllerImpl) ForgotPassword(c *gin.Context) {
	var body dto.ForgotPasswordReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	if err := a.AuthService.ForgotPassword(c.Request.Context(), &body); err != nil {
		switch {
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrCooldown):
			c.JSON(http.StatusTooManyRequests, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "password reset link has been sent"))

}

func(a *authControllerImpl) ResetPassword(c *gin.Context){
	var body dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	if err := a.AuthService.ResetPassword(c.Request.Context(), &body); err != nil{
		switch {
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrTokenInvalid):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success reset password"))

}

func(a *authControllerImpl) ChangePassword(c *gin.Context){
	var body dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)

	if err := a.AuthService.ChangePassword(c.Request.Context(), user.UserID, &body); err != nil {
		switch {
		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrPasswordSame):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		case errors.Is(err, service.ErrCredentialInvalid):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success change password"))

}
