package handler

import (
	"net/http"

	"whwriter/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	genreRepo    repository.GenreRepository
	platformRepo repository.PlatformRepository
	llmRepo      repository.LLMConfigRepository
}

func NewResourceHandler(genreRepo repository.GenreRepository, platformRepo repository.PlatformRepository, llmRepo repository.LLMConfigRepository) *ResourceHandler {
	return &ResourceHandler{
		genreRepo:    genreRepo,
		platformRepo: platformRepo,
		llmRepo:      llmRepo,
	}
}

func (h *ResourceHandler) ListGenres(c *gin.Context) {
	userID := c.GetUint("user_id")
	genres, err := h.genreRepo.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题材列表失败"})
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *ResourceHandler) ListPlatforms(c *gin.Context) {
	platforms, err := h.platformRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取平台列表失败"})
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *ResourceHandler) ListLLMConfigs(c *gin.Context) {
	configs, err := h.llmRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型配置失败"})
		return
	}
	c.JSON(http.StatusOK, configs)
}
