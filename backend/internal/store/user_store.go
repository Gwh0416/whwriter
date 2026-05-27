package store

import (
	"time"

	"whwriter/backend/pkg/models"

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

func (s *UserStore) UpdatePassword(userID uint, passwordHash string) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

func (s *UserStore) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	s.db.Model(&models.User{}).Count(&total)

	offset := (page - 1) * pageSize
	err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

type DashboardStats struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveBooks   int64 `json:"active_books"`
	TotalChapters int64 `json:"total_chapters"`
}

func (s *UserStore) GetStats() (*DashboardStats, error) {
	var stats DashboardStats

	s.db.Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.Model(&models.Book{}).Where("status = ?", models.BookStatusActive).Count(&stats.ActiveBooks)
	s.db.Model(&models.Chapter{}).Count(&stats.TotalChapters)

	return &stats, nil
}
