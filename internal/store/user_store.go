package store

import (
	"time"

	"whwriter/pkg/models"

	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *UserStore) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.db.Where("username = ?", username).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) SaveVerificationCode(email, code string) error {
	v := &models.EmailVerification{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return s.db.Create(v).Error
}

func (s *UserStore) VerifyCode(email, code string) (bool, error) {
	var v models.EmailVerification
	err := s.db.Where("email = ? AND code = ?", email, code).
		Order("id DESC").First(&v).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if v.Used || time.Now().After(v.ExpiresAt) {
		return false, nil
	}
	s.db.Model(&v).Update("used", true)
	return true, nil
}
