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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计数据失败"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取题材列表失败"})
		return
	}
	c.JSON(http.StatusOK, genres)
}

func (h *AdminHandler) CreateGenre(c *gin.Context) {
	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.genreRepo.Create(&genre); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建题材失败"})
		return
	}
	c.JSON(http.StatusOK, genre)
}

func (h *AdminHandler) UpdateGenre(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var genre model.Genre
	if err := c.ShouldBindJSON(&genre); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	genre.ID = uint(id)
	if err := h.genreRepo.Update(&genre); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新题材失败"})
		return
	}
	c.JSON(http.StatusOK, genre)
}

func (h *AdminHandler) DeleteGenre(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.genreRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除题材失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	status := model.UserStatus(body.Status)
	if status != model.UserStatusActive && status != model.UserStatusDisabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态值，仅允许 active 或 disabled"})
		return
	}

	if err := h.userRepo.UpdateStatus(uint(id), status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "状态更新成功"})
}

func (h *AdminHandler) AddBalance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if body.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "金额不能为0"})
		return
	}

	if err := h.userRepo.AddBalance(uint(id), body.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "充值失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "充值成功"})
}

func (h *AdminHandler) ListPlatforms(c *gin.Context) {
	platforms, err := h.platformRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取平台列表失败"})
		return
	}
	c.JSON(http.StatusOK, platforms)
}

func (h *AdminHandler) CreatePlatform(c *gin.Context) {
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := h.platformRepo.Create(&platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建平台失败"})
		return
	}
	c.JSON(http.StatusOK, platform)
}

func (h *AdminHandler) UpdatePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var platform model.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	platform.ID = uint(id)
	if err := h.platformRepo.Update(&platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新平台失败"})
		return
	}
	c.JSON(http.StatusOK, platform)
}

func (h *AdminHandler) DeletePlatform(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.platformRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除平台失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
