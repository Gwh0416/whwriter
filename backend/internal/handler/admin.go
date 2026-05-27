package handler

import (
	"net/http"
	"strconv"

	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/mysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userRepo repository.UserRepository
	db       *gorm.DB
}

func NewAdminHandler(userRepo repository.UserRepository, db *gorm.DB) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, db: db}
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
