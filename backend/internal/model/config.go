package model

import "time"

type LLMConfig struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Provider        string    `json:"provider" gorm:"size:128;not null"`
	Label           string    `json:"label" gorm:"size:128"`
	BaseURL         string    `json:"base_url" gorm:"size:512"`
	APIKeyEncrypted string    `json:"-" gorm:"size:512"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Models []LLMModel `json:"models,omitempty" gorm:"foreignKey:LLMConfigID"`
}

func (LLMConfig) TableName() string {
	return "llm_configs"
}

type LLMModel struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	LLMConfigID uint      `json:"llm_config_id" gorm:"index;not null"`
	ModelName   string    `json:"model_name" gorm:"size:128;not null"`
	IsEnabled   bool      `json:"is_enabled" gorm:"default:true"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	TokenUsage int64 `json:"token_usage" gorm:"-"`
}

func (LLMModel) TableName() string {
	return "llm_models"
}

type TokenUsage struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	LLMModelID       uint      `json:"llm_model_id" gorm:"index;not null"`
	PromptTokens     int64     `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int64     `json:"completion_tokens" gorm:"default:0"`
	TotalTokens      int64     `json:"total_tokens" gorm:"default:0"`
	CreatedAt        time.Time `json:"created_at"`
}

func (TokenUsage) TableName() string {
	return "token_usages"
}

type AgentModelRoute struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	BookID     uint      `json:"book_id" gorm:"uniqueIndex:idx_book_agent;not null"`
	AgentName  string    `json:"agent_name" gorm:"uniqueIndex:idx_book_agent;size:64;not null"`
	LLMModelID uint      `json:"llm_model_id" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	LLMModel LLMModel `json:"llm_model" gorm:"foreignKey:LLMModelID"`
}

func (AgentModelRoute) TableName() string {
	return "agent_model_routes"
}
