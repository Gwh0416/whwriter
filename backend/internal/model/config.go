package model

import "time"

type LLMConfig struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Provider        string    `json:"provider" gorm:"size:64;not null"`
	Service         string    `json:"service" gorm:"size:64"`
	BaseURL         string    `json:"base_url" gorm:"size:512"`
	APIKeyEncrypted string    `json:"-" gorm:"size:512"`
	Model           string    `json:"model" gorm:"size:128"`
	Temperature     float64   `json:"temperature" gorm:"type:decimal(3,2);default:0.70"`
	MaxTokens       uint      `json:"max_tokens" gorm:"default:4096"`
	IsDefault       bool      `json:"is_default" gorm:"default:false"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (LLMConfig) TableName() string {
	return "llm_configs"
}

type AgentModelRoute struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BookID      uint      `json:"book_id" gorm:"uniqueIndex:idx_book_agent;not null"`
	AgentName   string    `json:"agent_name" gorm:"uniqueIndex:idx_book_agent;size:64;not null"`
	LLMConfigID uint      `json:"llm_config_id" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	LLMConfig LLMConfig `json:"llm_config" gorm:"foreignKey:LLMConfigID"`
}

func (AgentModelRoute) TableName() string {
	return "agent_model_routes"
}
