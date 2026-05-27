package mysql

import (
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
)

type platformRepo struct {
	db *gorm.DB
}

func NewPlatformRepo(db *gorm.DB) repository.PlatformRepository {
	return &platformRepo{db: db}
}

func (r *platformRepo) List() ([]model.Platform, error) {
	var platforms []model.Platform
	err := r.db.Where("is_active = ?", true).Order("sort_order").Find(&platforms).Error
	return platforms, err
}
