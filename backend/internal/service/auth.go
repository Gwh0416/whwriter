package service

import (
	"errors"
	"time"
	"unicode"

	"whwriter/backend/internal/config"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists    = errors.New("邮箱已被注册")
	ErrUsernameAlreadyExists = errors.New("用户名已被占用")
	ErrInvalidCode           = errors.New("验证码无效或已过期")
	ErrInvalidCredentials    = errors.New("邮箱或密码错误")
	ErrWeakPassword          = errors.New("密码强度不足：需要至少8位，包含大小写字母和数字")
	ErrInvalidUsername       = errors.New("用户名需为2-16个字符")
)

type AuthService struct {
	userRepo repository.UserRepository
	emailSvc *EmailService
	cfg      *config.Config
}

func NewAuthService(userRepo repository.UserRepository, emailSvc *EmailService, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, emailSvc: emailSvc, cfg: cfg}
}

func (s *AuthService) SendCode(req *model.SendCodeRequest) error {
	code := s.emailSvc.GenerateCode()
	if err := s.userRepo.SaveVerificationCode(req.Email, code); err != nil {
		return err
	}
	return s.emailSvc.SendVerificationCode(req.Email, code)
}

func (s *AuthService) Register(req *model.RegisterRequest) (*model.AuthResponse, error) {
	if !validatePassword(req.Password) {
		return nil, ErrWeakPassword
	}

	if !validateUsername(req.Username) {
		return nil, ErrInvalidUsername
	}

	ok, err := s.userRepo.VerifyCode(req.Email, req.Code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCode
	}

	existing, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	existing, err = s.userRepo.FindByUsername(req.Username)
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

	user := &model.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hash),
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}, nil
}

func (s *AuthService) Login(req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
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

	return &model.AuthResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}, nil
}

func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
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

func validateUsername(username string) bool {
	count := len([]rune(username))
	return count >= 2 && count <= 16
}

func (s *AuthService) SendChangePasswordCode(email string) error {
	code := s.emailSvc.GenerateCode()
	if err := s.userRepo.SaveVerificationCode(email, code); err != nil {
		return err
	}
	return s.emailSvc.SendVerificationCode(email, code)
}

func (s *AuthService) ChangePassword(userID uint, req *model.ChangePasswordRequest) error {
	if !validatePassword(req.NewPassword) {
		return ErrWeakPassword
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return err
	}

	ok, err := s.userRepo.VerifyCode(user.Email, req.Code)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCode
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, string(hash))
}
