package handler

import (
	"net/http"

	"whwriter/backend/internal/store"
	"whwriter/backend/pkg/models"

	"github.com/gin-gonic/gin"
)

type BookHandler struct {
	bookStore *store.BookStore
}

func NewBookHandler(bookStore *store.BookStore) *BookHandler {
	return &BookHandler{bookStore: bookStore}
}

func (h *BookHandler) Create(c *gin.Context) {
	var req models.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整的书籍信息"})
		return
	}

	userID := c.GetUint("user_id")

	book := &models.Book{
		UserID:           userID,
		Title:            req.Title,
		GenreID:          req.GenreID,
		PlatformID:       req.PlatformID,
		ChapterWordCount: req.ChapterWordCount,
		TargetChapters:   req.TargetChapters,
		Status:           models.BookStatusOutlining,
		AutomationMode:   models.AutomationSemi,
	}

	if err := h.bookStore.Create(book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建书籍失败"})
		return
	}

	c.JSON(http.StatusCreated, book)
}

func (h *BookHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	books, err := h.bookStore.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取书籍列表失败"})
		return
	}
	c.JSON(http.StatusOK, books)
}
