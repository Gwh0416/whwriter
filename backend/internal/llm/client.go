package llm

import (
	"context"
	"fmt"
	"time"

	"whwriter/backend/internal/config"
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type Client struct {
	llmModelRepo   repository.LLMModelRepository
	llmConfigRepo  repository.LLMConfigRepository
	tokenUsageRepo repository.TokenUsageRepository
	defaultTimeout time.Duration
	agentTimeouts  map[string]time.Duration
}

func NewClient(llmModelRepo repository.LLMModelRepository, llmConfigRepo repository.LLMConfigRepository, tokenUsageRepo repository.TokenUsageRepository, cfg config.LLMConfig) *Client {
	defaultSeconds := cfg.DefaultTimeoutSeconds
	if defaultSeconds <= 0 {
		defaultSeconds = 120
	}
	return &Client{
		llmModelRepo:   llmModelRepo,
		llmConfigRepo:  llmConfigRepo,
		tokenUsageRepo: tokenUsageRepo,
		defaultTimeout: time.Duration(defaultSeconds) * time.Second,
		agentTimeouts: map[string]time.Duration{
			"planner": timeoutDuration(cfg.PlannerTimeoutSeconds, defaultSeconds),
			"writer":  timeoutDuration(cfg.WriterTimeoutSeconds, defaultSeconds),
			"settler": timeoutDuration(cfg.SettlerTimeoutSeconds, defaultSeconds),
			"auditor": timeoutDuration(cfg.AuditorTimeoutSeconds, defaultSeconds),
			"reviser": timeoutDuration(cfg.ReviserTimeoutSeconds, defaultSeconds),
		},
	}
}

type AgentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *Client) Chat(ctx context.Context, modelID uint, systemPrompt string, messages []AgentMessage, temperature float64) (string, error) {
	return c.chatWithTimeout(ctx, "", modelID, systemPrompt, messages, temperature)
}

func (c *Client) ChatForAgent(ctx context.Context, agentName string, modelID uint, systemPrompt string, messages []AgentMessage, temperature float64) (string, error) {
	return c.chatWithTimeout(ctx, agentName, modelID, systemPrompt, messages, temperature)
}

func (c *Client) chatWithTimeout(ctx context.Context, agentName string, modelID uint, systemPrompt string, messages []AgentMessage, temperature float64) (string, error) {
	if modelID == 0 {
		return "", fmt.Errorf("model not configured")
	}

	llmModel, err := c.llmModelRepo.FindByID(modelID)
	if err != nil {
		return "", fmt.Errorf("model not found: %w", err)
	}

	cfg, err := c.llmConfigRepo.FindByID(llmModel.LLMConfigID)
	if err != nil {
		return "", fmt.Errorf("config not found: %w", err)
	}

	t := float32(temperature)
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     cfg.BaseURL,
		APIKey:      cfg.APIKeyEncrypted,
		Model:       llmModel.ModelName,
		Temperature: &t,
	})
	if err != nil {
		return "", fmt.Errorf("create chat model: %w", err)
	}

	var msgs []*schema.Message
	msgs = append(msgs, schema.SystemMessage(systemPrompt))
	for _, m := range messages {
		switch m.Role {
		case "user":
			msgs = append(msgs, schema.UserMessage(m.Content))
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
		}
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeoutForAgent(agentName))
	defer cancel()

	resp, err := chatModel.Generate(callCtx, msgs)
	if err != nil {
		return "", fmt.Errorf("generate: %w", err)
	}
	c.recordUsage(llmModel.ID, resp)

	return resp.Content, nil
}

func (c *Client) recordUsage(modelID uint, resp *schema.Message) {
	if c.tokenUsageRepo == nil || resp == nil || resp.ResponseMeta == nil || resp.ResponseMeta.Usage == nil {
		return
	}

	usage := resp.ResponseMeta.Usage
	if usage.TotalTokens <= 0 && usage.PromptTokens <= 0 && usage.CompletionTokens <= 0 {
		return
	}

	_ = c.tokenUsageRepo.Record(&model.TokenUsage{
		LLMModelID:       modelID,
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
	})
}

func (c *Client) timeoutForAgent(agentName string) time.Duration {
	if d, ok := c.agentTimeouts[agentName]; ok && d > 0 {
		return d
	}
	return c.defaultTimeout
}

func timeoutDuration(seconds int, fallback int) time.Duration {
	if seconds <= 0 {
		seconds = fallback
	}
	return time.Duration(seconds) * time.Second
}

func (c *Client) GetDefaultModel() (*model.LLMModel, error) {
	models, err := c.llmModelRepo.ListEnabled()
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		if m.IsDefault {
			return &m, nil
		}
	}
	if len(models) > 0 {
		return &models[0], nil
	}
	return nil, fmt.Errorf("no enabled models")
}
