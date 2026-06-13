package mysql

import (
	"time"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) repository.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepo) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) SaveVerificationCode(email, code string) error {
	v := &model.EmailVerification{
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return r.db.Create(v).Error
}

func (r *userRepo) VerifyCode(email, code string) (bool, error) {
	var v model.EmailVerification
	err := r.db.Where("email = ? AND code = ?", email, code).
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
	r.db.Model(&v).Update("used", true)
	return true, nil
}

func (r *userRepo) UpdatePassword(userID uint, passwordHash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

func (r *userRepo) ListUsers(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	r.db.Model(&model.User{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *userRepo) GetStats() (*repository.DashboardStats, error) {
	var stats repository.DashboardStats

	r.db.Model(&model.User{}).Count(&stats.TotalUsers)
	r.db.Model(&model.Book{}).Where("status = ?", model.BookStatusActive).Count(&stats.ActiveBooks)
	r.db.Model(&model.Chapter{}).Count(&stats.TotalChapters)

	return &stats, nil
}

func (r *userRepo) UpdateStatus(userID uint, status model.UserStatus) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

func (r *userRepo) AddBalance(userID uint, amount int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("balance", gorm.Expr("balance + ?", amount)).Error
}
