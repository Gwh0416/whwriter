package handler

import (
	"net/http"
	"strconv"

	"whwriter/backend/internal/model"
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

func (h *ResourceHandler) ListMyGenres(c *gin.Context) {
	userID := c.GetUint("user_id")
	genres, err := h.genreRepo.ListMyGenres(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题材列表失败"})
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *ResourceHandler) CreateMyGenre(c *gin.Context) {
	userID := c.GetUint("user_id")
	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	genre.UserID = userID
	genre.ID = 0
	if err := h.genreRepo.Create(&genre); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题材失败"})
		return
	}
	c.JSON(http.StatusCreated, genre)
}

func (h *ResourceHandler) UpdateMyGenre(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	existing, err := h.genreRepo.FindByID(uint(id))
	if err != nil || existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该题材"})
		return
	}

	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	genre.ID = uint(id)
	genre.UserID = userID
	if err := h.genreRepo.Update(&genre); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题材失败"})
		return
	}
	c.JSON(http.StatusOK, genre)
}

func (h *ResourceHandler) DeleteMyGenre(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	existing, err := h.genreRepo.FindByID(uint(id))
	if err != nil || existing.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该题材"})
		return
	}

	if err := h.genreRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题材失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
