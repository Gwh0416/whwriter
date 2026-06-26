package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"whwriter/backend/internal/model"

	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/mysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userRepo      repository.UserRepository
	genreRepo     repository.GenreRepository
	platformRepo  repository.PlatformRepository
	llmConfigRepo repository.LLMConfigRepository
	llmModelRepo  repository.LLMModelRepository
	db            *gorm.DB
}

func NewAdminHandler(userRepo repository.UserRepository, genreRepo repository.GenreRepository, platformRepo repository.PlatformRepository, llmConfigRepo repository.LLMConfigRepository, llmModelRepo repository.LLMModelRepository, db *gorm.DB) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, genreRepo: genreRepo, platformRepo: platformRepo, llmConfigRepo: llmConfigRepo, llmModelRepo: llmModelRepo, db: db}
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.userRepo.GetStats()
	if err != nil {
		ErrStatsFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.userRepo.ListUsers(page, pageSize)
	if err != nil {
		ErrUserListFailed.JSON(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AdminHandler) Initialize(c *gin.Context) {
	mysql.SeedGenres(h.db)
	mysql.SeedPlatforms(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "基础数据初始化完成"})
}

func (h *AdminHandler) ListGenres(c *gin.Context) {
	genres, err := h.genreRepo.ListAll()
	if err != nil {
		ErrGenreListFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *AdminHandler) CreateGenre(c *gin.Context) {
	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	if err := h.genreRepo.Create(&genre); err != nil {
		ErrGenreCreateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, genre)
}

func (h *AdminHandler) UpdateGenre(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	genre.ID = uint(id)
	if err := h.genreRepo.Update(&genre); err != nil {
		ErrGenreUpdateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, genre)
}

func (h *AdminHandler) DeleteGenre(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.genreRepo.Delete(uint(id)); err != nil {
		ErrGenreDeleteFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}

	status := model.UserStatus(body.Status)
	if status != model.UserStatusActive && status != model.UserStatusDisabled {
		ErrInvalidStatus.JSON(c)
		return
	}

	if err := h.userRepo.UpdateStatus(uint(id), status); err != nil {
		ErrUpdateStatusFailed.JSON(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "状态更新成功"})
}

func (h *AdminHandler) AddBalance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}

	if body.Amount == 0 {
		ErrAmountZero.JSON(c)
		return
	}

	if err := h.userRepo.AddBalance(uint(id), body.Amount); err != nil {
		ErrRechargeFailed.JSON(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "充值成功"})
}

func (h *AdminHandler) ListPlatforms(c *gin.Context) {
	platforms, err := h.platformRepo.ListAll()
	if err != nil {
		ErrPlatformListFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *AdminHandler) CreatePlatform(c *gin.Context) {
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	if err := h.platformRepo.Create(&platform); err != nil {
		ErrPlatformCreateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, platform)
}

func (h *AdminHandler) UpdatePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	platform.ID = uint(id)
	if err := h.platformRepo.Update(&platform); err != nil {
		ErrPlatformUpdateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, platform)
}

func (h *AdminHandler) DeletePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.platformRepo.Delete(uint(id)); err != nil {
		ErrPlatformDeleteFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *AdminHandler) ListLLMConfigs(c *gin.Context) {
	configs, err := h.llmConfigRepo.ListAllWithModels()
	if err != nil {
		ErrLLMConfigFailed.JSON(c)
		return
	}

	usageMap, err := h.llmModelRepo.GetTokenUsageByModel()
	if err != nil {
		usageMap = make(map[uint]int64)
	}

	for i := range configs {
		for j := range configs[i].Models {
			if usage, ok := usageMap[configs[i].Models[j].ID]; ok {
				configs[i].Models[j].TokenUsage = usage
			}
		}
	}

	c.JSON(http.StatusOK, configs)
}

func (h *AdminHandler) CreateLLMConfig(c *gin.Context) {
	var body struct {
		Provider string `json:"provider"`
		Label    string `json:"label"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	config := model.LLMConfig{
		Provider:        body.Provider,
		Label:           body.Label,
		BaseURL:         body.BaseURL,
		APIKeyEncrypted: body.APIKey,
	}
	if err := h.llmConfigRepo.Create(&config); err != nil {
		ErrLLMCreateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *AdminHandler) UpdateLLMConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	existing, err := h.llmConfigRepo.FindByID(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Label    string `json:"label"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}

	existing.Provider = body.Provider
	existing.Label = body.Label
	existing.BaseURL = body.BaseURL
	if body.APIKey != "" {
		existing.APIKeyEncrypted = body.APIKey
	}

	if err := h.llmConfigRepo.Update(existing); err != nil {
		ErrLLMUpdateFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *AdminHandler) DeleteLLMConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.llmModelRepo.DeleteByConfig(uint(id)); err != nil {
		ErrLLMDeleteFailed.JSON(c)
		return
	}
	if err := h.llmConfigRepo.Delete(uint(id)); err != nil {
		ErrLLMDeleteFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *AdminHandler) TestLLMConnection(c *gin.Context) {
	var body struct {
		ConfigID uint   `json:"config_id"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}

	apiKey := body.APIKey
	if apiKey == "" && body.ConfigID > 0 {
		cfg, err := h.llmConfigRepo.FindByID(body.ConfigID)
		if err == nil {
			apiKey = cfg.APIKeyEncrypted
		}
	}

	baseURL := strings.TrimRight(body.BaseURL, "/")
	req, err := http.NewRequest("GET", baseURL+"/models", nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "URL格式错误"})
		return
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "连接失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": fmt.Sprintf("返回状态码 %d", resp.StatusCode)})
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "读取响应失败"})
		return
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": "解析模型列表失败"})
		return
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"models":  models,
	})
}

func (h *AdminHandler) SaveLLMModels(c *gin.Context) {
	configID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	var body struct {
		Models []struct {
			ModelName string `json:"model_name"`
			IsEnabled bool   `json:"is_enabled"`
			IsDefault bool   `json:"is_default"`
		} `json:"models"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		ErrBadRequest.JSON(c)
		return
	}

	if err := h.llmModelRepo.DeleteByConfig(uint(configID)); err != nil {
		ErrLLMUpdateFailed.JSON(c)
		return
	}

	for _, m := range body.Models {
		model := model.LLMModel{
			LLMConfigID: uint(configID),
			ModelName:   m.ModelName,
			IsEnabled:   m.IsEnabled,
			IsDefault:   m.IsDefault,
		}
		if err := h.llmModelRepo.Create(&model); err != nil {
			ErrLLMUpdateFailed.JSON(c)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func (h *AdminHandler) SetDefaultModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.llmModelRepo.SetDefault(uint(id)); err != nil {
		ErrLLMDefaultFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已设为默认模型"})
}

func (h *AdminHandler) GetTokenUsageStats(c *gin.Context) {
	configs, err := h.llmConfigRepo.ListAllWithModels()
	if err != nil {
		ErrLLMUsageFailed.JSON(c)
		return
	}

	usageMap, err := h.llmModelRepo.GetTokenUsageByModel()
	if err != nil {
		ErrLLMUsageFailed.JSON(c)
		return
	}

	totalUsage, err := h.llmModelRepo.GetTotalTokenUsage()
	if err != nil {
		ErrLLMUsageFailed.JSON(c)
		return
	}

	type modelUsage struct {
		ID         uint   `json:"id"`
		ConfigID   uint   `json:"config_id"`
		Provider   string `json:"provider"`
		ModelName  string `json:"model_name"`
		TokenUsage int64  `json:"token_usage"`
	}

	var details []modelUsage
	for _, cfg := range configs {
		for _, m := range cfg.Models {
			details = append(details, modelUsage{
				ID:         m.ID,
				ConfigID:   cfg.ID,
				Provider:   cfg.Label,
				ModelName:  m.ModelName,
				TokenUsage: usageMap[m.ID],
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_usage": totalUsage,
		"details":     details,
	})
}
