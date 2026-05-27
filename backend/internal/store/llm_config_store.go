package store

import (
	"whwriter/backend/pkg/models"

	"gorm.io/gorm"
)

type LLMConfigStore struct {
	db *gorm.DB
}

func NewLLMConfigStore(db *gorm.DB) *LLMConfigStore {
	return &LLMConfigStore{db: db}
}

func (s *LLMConfigStore) List() ([]models.LLMConfig, error) {
	var configs []models.LLMConfig
	err := s.db.Order("id").Find(&configs).Error
	return configs, err
}
