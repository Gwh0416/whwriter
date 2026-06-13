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

func (r *llmConfigRepo) ListAll() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	err := r.db.Order("id").Find(&configs).Error
	return configs, err
}

func (r *llmConfigRepo) ListAllWithModels() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	err := r.db.Preload("Models").Order("id").Find(&configs).Error
	return configs, err
}

func (r *llmConfigRepo) FindByID(id uint) (*model.LLMConfig, error) {
	var config model.LLMConfig
	err := r.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *llmConfigRepo) Create(config *model.LLMConfig) error {
	return r.db.Create(config).Error
}

func (r *llmConfigRepo) Update(config *model.LLMConfig) error {
	return r.db.Save(config).Error
}

func (r *llmConfigRepo) Delete(id uint) error {
	return r.db.Delete(&model.LLMConfig{}, id).Error
}

type llmModelRepo struct {
	db *gorm.DB
}

func NewLLMModelRepo(db *gorm.DB) repository.LLMModelRepository {
	return &llmModelRepo{db: db}
}

func (r *llmModelRepo) ListByConfig(configID uint) ([]model.LLMModel, error) {
	var models []model.LLMModel
	err := r.db.Where("llm_config_id = ?", configID).Order("id").Find(&models).Error
	return models, err
}

func (r *llmModelRepo) ListEnabled() ([]model.LLMModel, error) {
	var models []model.LLMModel
	err := r.db.Where("is_enabled = ?", true).Order("id").Find(&models).Error
	return models, err
}

func (r *llmModelRepo) FindByID(id uint) (*model.LLMModel, error) {
	var m model.LLMModel
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *llmModelRepo) Create(m *model.LLMModel) error {
	return r.db.Create(m).Error
}

func (r *llmModelRepo) Update(m *model.LLMModel) error {
	return r.db.Save(m).Error
}

func (r *llmModelRepo) Delete(id uint) error {
	return r.db.Delete(&model.LLMModel{}, id).Error
}

func (r *llmModelRepo) DeleteByConfig(configID uint) error {
	return r.db.Where("llm_config_id = ?", configID).Delete(&model.LLMModel{}).Error
}

func (r *llmModelRepo) SetDefault(modelID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.LLMModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.LLMModel{}).Where("id = ?", modelID).Update("is_default", true).Error
	})
}

func (r *llmModelRepo) GetTokenUsage(modelID uint) (int64, error) {
	var total int64
	err := r.db.Model(&model.TokenUsage{}).Where("llm_model_id = ?", modelID).Select("COALESCE(SUM(total_tokens), 0)").Scan(&total).Error
	return total, err
}

func (r *llmModelRepo) GetTotalTokenUsage() (int64, error) {
	var total int64
	err := r.db.Model(&model.TokenUsage{}).Select("COALESCE(SUM(total_tokens), 0)").Scan(&total).Error
	return total, err
}

func (r *llmModelRepo) GetTokenUsageByModel() (map[uint]int64, error) {
	type row struct {
		LLMModelID uint
		Total      int64
	}
	var rows []row
	err := r.db.Model(&model.TokenUsage{}).Select("llm_model_id, SUM(total_tokens) as total").Group("llm_model_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]int64)
	for _, r := range rows {
		result[r.LLMModelID] = r.Total
	}
	return result, nil
}

type tokenUsageRepo struct {
	db *gorm.DB
}

func NewTokenUsageRepo(db *gorm.DB) repository.TokenUsageRepository {
	return &tokenUsageRepo{db: db}
}

func (r *tokenUsageRepo) Record(usage *model.TokenUsage) error {
	return r.db.Create(usage).Error
}
