package controller

import (
	"errors"
	"net/http"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/response"
	"user-auth-exam/internal/service"

	"github.com/gin-gonic/gin"
)

type UserController interface {
	Update(c *gin.Context)
	Deactive(c *gin.Context)
	GetByID(c *gin.Context)
	ChangeEmail(c *gin.Context)
}

type userControllerImpl struct {
	UserService service.UserService
}

func NewUserControllerImpl(userService service.UserService) UserController {
	return &userControllerImpl{
		UserService: userService,
	}
}

func (u *userControllerImpl) Update(c *gin.Context) {
	var req dto.UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	//get user from claim token
	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)
	req.ID = user.UserID

	err := u.UserService.Update(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success"))

}

func (u *userControllerImpl) Deactive(c *gin.Context) {
	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)

	if err := u.UserService.Deactivate(c.Request.Context(), user.UserID); err != nil {
		if errors.Is(err, service.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success"))

}

func (u *userControllerImpl) GetByID(c *gin.Context) {
	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)

	resp, err := u.UserService.GetByID(c.Request.Context(), user.UserID)
	if err != nil {
		if errors.Is(err, service.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessResponse(resp, "success"))
}

func (u *userControllerImpl) ChangeEmail(c *gin.Context) {
	claim, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorized"))
		return
	}

	user := claim.(*dto.TokenClaim)

	var body dto.UpdateEmailRequest
	body.ID = user.UserID
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse("invalid request body"))
		return
	}

	if err := u.UserService.ChangeEmail(c.Request.Context(), &body); err != nil {
		switch {
		case errors.Is(err, service.ErrEmailExist):
			c.JSON(http.StatusConflict, response.ErrorResponse(err.Error()))

		case errors.Is(err, service.ErrValidationFailed):
			c.JSON(http.StatusBadRequest, response.ErrorResponse(err.Error()))

		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse(err.Error()))
		}

		return

	}

	c.JSON(http.StatusOK, response.SuccessResponse(nil, "success"))
}
