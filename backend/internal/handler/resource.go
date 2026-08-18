package handler

import (
	"net/http"

	"whwriter/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	genreRepo     repository.GenreRepository
	platformRepo  repository.PlatformRepository
	llmModelRepo  repository.LLMModelRepository
	llmConfigRepo repository.LLMConfigRepository
}

func NewResourceHandler(genreRepo repository.GenreRepository, platformRepo repository.PlatformRepository, llmModelRepo repository.LLMModelRepository, llmConfigRepo repository.LLMConfigRepository) *ResourceHandler {
	return &ResourceHandler{
		genreRepo:     genreRepo,
		platformRepo:  platformRepo,
		llmModelRepo:  llmModelRepo,
		llmConfigRepo: llmConfigRepo,
	}
}

func (h *ResourceHandler) ListGenres(c *gin.Context) {
	genres, err := h.genreRepo.ListAll()
	if err != nil {
		ErrGenreListFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *ResourceHandler) ListPlatforms(c *gin.Context) {
	platforms, err := h.platformRepo.List()
	if err != nil {
		ErrPlatformListFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *ResourceHandler) ListLLMConfigs(c *gin.Context) {
	configs, err := h.llmConfigRepo.ListAllWithModels()
	if err != nil {
		ErrLLMConfigFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, configs)
}
