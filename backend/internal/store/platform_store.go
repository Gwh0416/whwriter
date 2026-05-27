package store

import (
	"whwriter/backend/pkg/models"

	"gorm.io/gorm"
)

type PlatformStore struct {
	db *gorm.DB
}

func NewPlatformStore(db *gorm.DB) *PlatformStore {
	return &PlatformStore{db: db}
}

func (s *PlatformStore) List() ([]models.Platform, error) {
	var platforms []models.Platform
	err := s.db.Where("is_active = ?", true).Order("sort_order").Find(&platforms).Error
	return platforms, err
}
