package mysql

import (
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
)

type llmConfigRepo struct {
	db *gorm.DB
}

func NewLLMConfigRepo(db *gorm.DB) repository.LLMConfigRepository {
	return &llmConfigRepo{db: db}
}

func (r *llmConfigRepo) List() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	err := r.db.Order("id").Find(&configs).Error
	return configs, err
}
