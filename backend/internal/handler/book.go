package handler

import (
	"net/http"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type BookHandler struct {
	bookRepo repository.BookRepository
}

func NewBookHandler(bookRepo repository.BookRepository) *BookHandler {
	return &BookHandler{bookRepo: bookRepo}
}

func (h *BookHandler) Create(c *gin.Context) {
	var req model.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrIncompleteBook.JSON(c)
		return
	}

	userID := c.GetUint("user_id")

	book := &model.Book{
		UserID:           userID,
		Title:            req.Title,
		GenreID:          req.GenreID,
		PlatformID:       req.PlatformID,
		ChapterWordCount: req.ChapterWordCount,
		TargetChapters:   req.TargetChapters,
		Status:           model.BookStatusOutlining,
		AutomationMode:   model.AutomationSemi,
	}

	if err := h.bookRepo.Create(book); err != nil {
		ErrBookCreateFailed.JSON(c)
		return
	}

	c.JSON(http.StatusCreated, book)
}

func (h *BookHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	books, err := h.bookRepo.ListByUser(userID)
	if err != nil {
		ErrBookListFailed.JSON(c)
		return
	}
	c.JSON(http.StatusOK, books)
}
