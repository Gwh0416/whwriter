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

func (s *LLMConfigStore) ListByUser(userID uint) ([]models.LLMConfig, error) {
	var configs []models.LLMConfig
	err := s.db.Where("user_id = ?", userID).Order("id").Find(&configs).Error
	return configs, err
}
