package handler

import (
	"net/http"

	"whwriter/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	genreStore    *store.GenreStore
	platformStore *store.PlatformStore
	llmStore      *store.LLMConfigStore
}

func NewResourceHandler(genreStore *store.GenreStore, platformStore *store.PlatformStore, llmStore *store.LLMConfigStore) *ResourceHandler {
	return &ResourceHandler{
		genreStore:    genreStore,
		platformStore: platformStore,
		llmStore:      llmStore,
	}
}

func (h *ResourceHandler) ListGenres(c *gin.Context) {
	userID := c.GetUint("user_id")
	genres, err := h.genreStore.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题材列表失败"})
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *ResourceHandler) ListPlatforms(c *gin.Context) {
	platforms, err := h.platformStore.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取平台列表失败"})
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *ResourceHandler) ListLLMConfigs(c *gin.Context) {
	userID := c.GetUint("user_id")
	configs, err := h.llmStore.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型配置失败"})
		return
	}
	c.JSON(http.StatusOK, configs)
}
