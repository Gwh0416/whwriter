package handler

import (
	"net/http"
	"strconv"

	"whwriter/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	userStore *store.UserStore
}

func NewAdminHandler(userStore *store.UserStore) *AdminHandler {
	return &AdminHandler{userStore: userStore}
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.userStore.GetStats()
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

	users, total, err := h.userStore.ListUsers(page, pageSize)
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
