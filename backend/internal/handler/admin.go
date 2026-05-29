package handler

import (
	"net/http"
	"strconv"
	"whwriter/backend/internal/model"

	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/mysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userRepo     repository.UserRepository
	genreRepo    repository.GenreRepository
	platformRepo repository.PlatformRepository
	db           *gorm.DB
}

func NewAdminHandler(userRepo repository.UserRepository, genreRepo repository.GenreRepository, platformRepo repository.PlatformRepository, db *gorm.DB) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, genreRepo: genreRepo, platformRepo: platformRepo, db: db}
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
	mysql.SeedAll(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "初始化完成"})
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
