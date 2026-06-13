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

func (r *platformRepo) ListAll() ([]model.Platform, error) {
	var platforms []model.Platform
	err := r.db.Order("sort_order").Find(&platforms).Error
	return platforms, err
}

func (r *platformRepo) Create(platform *model.Platform) error {
	return r.db.Create(platform).Error
}

func (r *platformRepo) Update(platform *model.Platform) error {
	return r.db.Save(platform).Error
}

func (r *platformRepo) Delete(id uint) error {
	return r.db.Delete(&model.Platform{}, id).Error
}
