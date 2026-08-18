package sqlite

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

func (r *llmModelRepo) GetTokenUsage(modelID uint) (model.TokenUsageSummary, error) {
	var summary model.TokenUsageSummary
	err := r.db.Model(&model.TokenUsage{}).
		Where("llm_model_id = ?", modelID).
		Select(`
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		`).
		Scan(&summary).Error
	return summary, err
}

func (r *llmModelRepo) GetTotalTokenUsage() (model.TokenUsageSummary, error) {
	var summary model.TokenUsageSummary
	err := r.db.Model(&model.TokenUsage{}).
		Select(`
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		`).
		Scan(&summary).Error
	return summary, err
}

func (r *llmModelRepo) GetTokenUsageByModel() (map[uint]model.TokenUsageSummary, error) {
	type row struct {
		LLMModelID       uint
		PromptTokens     int64
		CompletionTokens int64
		CachedTokens     int64
		TotalTokens      int64
	}
	var rows []row
	err := r.db.Model(&model.TokenUsage{}).
		Select(`
			llm_model_id,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		`).
		Group("llm_model_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]model.TokenUsageSummary)
	for _, r := range rows {
		result[r.LLMModelID] = model.TokenUsageSummary{
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			CachedTokens:     r.CachedTokens,
			TotalTokens:      r.TotalTokens,
		}
	}
	return result, nil
}

func (r *llmModelRepo) GetTokenUsageByAgent() ([]model.AgentTokenUsageSummary, error) {
	var rows []model.AgentTokenUsageSummary
	err := r.db.Model(&model.TokenUsage{}).
		Select(`
			COALESCE(NULLIF(agent_name, ''), 'chat') AS agent_name,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		`).
		Group("COALESCE(NULLIF(agent_name, ''), 'chat')").
		Order("total_tokens desc").
		Scan(&rows).Error
	return rows, err
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

func (r *tokenUsageRepo) LatestID() (uint, error) {
	var id uint
	err := r.db.Model(&model.TokenUsage{}).Select("COALESCE(MAX(id), 0)").Scan(&id).Error
	return id, err
}

func (r *tokenUsageRepo) SummaryAfterID(afterID uint) ([]model.AgentTokenUsageSummary, error) {
	var rows []model.AgentTokenUsageSummary
	err := r.db.Model(&model.TokenUsage{}).
		Where("id > ?", afterID).
		Select(`
			COALESCE(NULLIF(agent_name, ''), 'chat') AS agent_name,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		`).
		Group("COALESCE(NULLIF(agent_name, ''), 'chat')").
		Order("total_tokens desc").
		Scan(&rows).Error
	return rows, err
}
