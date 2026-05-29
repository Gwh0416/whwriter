package handler

import (
	"errors"
	"net/http"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) SendCode(c *gin.Context) {
	var req model.SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrInvalidEmail.JSON(c)
		return
	}

	if err := h.authSvc.SendCode(&req); err != nil {
		ErrSendCodeFailed.JSON(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrIncompleteRegister.JSON(c)
		return
	}

	resp, err := h.authSvc.Register(&req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWeakPassword),
			errors.Is(err, service.ErrInvalidCode),
			errors.Is(err, service.ErrEmailAlreadyExists),
			errors.Is(err, service.ErrUsernameAlreadyExists):
			ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		default:
			ErrRegisterFailed.JSON(c)
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrMissingCredentials.JSON(c)
		return
	}

	resp, err := h.authSvc.Login(&req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountDisabled) {
			ErrJSON(c, http.StatusUnauthorized, CodeInvalidCredentials, err.Error())
		} else {
			ErrLoginFailed.JSON(c)
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) SendChangePasswordCode(c *gin.Context) {
	role := c.GetString("role")
	if role == string(model.RoleAdmin) {
		ErrAdminNoPassword.JSON(c)
		return
	}

	email := c.GetString("email")
	if email == "" {
		ErrUnauthorized.JSON(c)
		return
	}

	if err := h.authSvc.SendChangePasswordCode(email); err != nil {
		ErrSendCodeFailed.JSON(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	role := c.GetString("role")
	if role == string(model.RoleAdmin) {
		ErrAdminNoPassword.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		ErrUnauthorized.JSON(c)
		return
	}

	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrIncompleteChangePwd.JSON(c)
		return
	}

	if err := h.authSvc.ChangePassword(userID, &req); err != nil {
		switch {
		case errors.Is(err, service.ErrWeakPassword),
			errors.Is(err, service.ErrInvalidCode):
			ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		default:
			ErrChangePwdFailed.JSON(c)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}
