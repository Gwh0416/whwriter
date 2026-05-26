package service

import (
	"errors"
	"time"
	"unicode"

	"whwriter/internal/config"
	"whwriter/internal/store"
	"whwriter/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists    = errors.New("邮箱已被注册")
	ErrUsernameAlreadyExists = errors.New("用户名已被占用")
	ErrInvalidCode           = errors.New("验证码无效或已过期")
	ErrInvalidCredentials    = errors.New("邮箱或密码错误")
	ErrWeakPassword          = errors.New("密码强度不足：需要至少8位，包含大小写字母和数字")
)

type AuthService struct {
	userStore *store.UserStore
	emailSvc  *EmailService
	cfg       *config.Config
}

func NewAuthService(userStore *store.UserStore, emailSvc *EmailService, cfg *config.Config) *AuthService {
	return &AuthService{userStore: userStore, emailSvc: emailSvc, cfg: cfg}
}

func (s *AuthService) SendCode(req *models.SendCodeRequest) error {
	code := s.emailSvc.GenerateCode()
	if err := s.userStore.SaveVerificationCode(req.Email, code); err != nil {
		return err
	}
	return s.emailSvc.SendVerificationCode(req.Email, code)
}

func (s *AuthService) Register(req *models.RegisterRequest) (*models.AuthResponse, error) {
	if !validatePassword(req.Password) {
		return nil, ErrWeakPassword
	}

	ok, err := s.userStore.VerifyCode(req.Email, req.Code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCode
	}

	existing, err := s.userStore.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	existing, err = s.userStore.FindByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hash),
	}
	if err := s.userStore.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (s *AuthService) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userStore.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func validatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}
