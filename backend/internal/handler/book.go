package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/pipeline"
	"whwriter/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type BookHandler struct {
	bookRepo  repository.BookRepository
	truthRepo repository.TruthFileRepository
	pipeline  *pipeline.Pipeline
}

func NewBookHandler(bookRepo repository.BookRepository, truthRepo repository.TruthFileRepository, pl *pipeline.Pipeline) *BookHandler {
	return &BookHandler{bookRepo: bookRepo, truthRepo: truthRepo, pipeline: pl}
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
		LLMModelID:       req.LLMModelID,
		Description:      req.Description,
		ChapterWordCount: req.ChapterWordCount,
		TargetChapters:   req.TargetChapters,
		Status:           model.BookStatusInitializing,
		AutomationMode:   model.AutomationSemi,
	}

	if err := h.bookRepo.Create(book); err != nil {
		ErrBookCreateFailed.JSON(c)
		return
	}

	if err := h.pipeline.InitBook(c.Request.Context(), pipeline.InitBookInput{BookID: book.ID}); err != nil {
		_ = h.truthRepo.UpdateBookStatus(book.ID, model.BookStatusPaused)
		ErrJSON(c, ErrBookInitFailed.StatusCode, ErrBookInitFailed.Code, ErrBookInitFailed.Message+"："+err.Error())
		return
	}

	if err := h.truthRepo.UpdateBookStatus(book.ID, model.BookStatusOutlining); err != nil {
		ErrBookInitFailed.JSON(c)
		return
	}

	initializedBook, err := h.truthRepo.GetBook(book.ID)
	if err != nil {
		ErrBookInitFailed.JSON(c)
		return
	}

	c.JSON(http.StatusCreated, initializedBook)
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

func (h *BookHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	chapters, _ := h.truthRepo.ListChapters(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"book":     book,
		"chapters": chapters,
	})
}

func (h *BookHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	if err := h.truthRepo.DeleteBookCascade(uint(id)); err != nil {
		ErrJSON(c, ErrBookDeleteFailed.StatusCode, ErrBookDeleteFailed.Code, ErrBookDeleteFailed.Message+"："+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "书籍及其关联数据已删除"})
}

func (h *BookHandler) WriteChapter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	if book.Status == model.BookStatusInitializing || book.Status == model.BookStatusWriting {
		ErrBookBusy.JSON(c)
		return
	}

	if book.Status == model.BookStatusPaused || book.Status == model.BookStatusCompleted {
		ErrJSON(c, http.StatusBadRequest, ErrBookWriteFailed.Code, "当前状态不允许继续写作")
		return
	}

	var body struct {
		ModelID   uint   `json:"model_id"`
		UserInput string `json:"user_input"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.ModelID = book.LLMModelID
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	locked, err := h.truthRepo.TransitionBookStatus(book.ID, []model.BookStatus{
		model.BookStatusOutlining,
		model.BookStatusActive,
	}, model.BookStatusWriting)
	if err != nil {
		errData, _ := json.Marshal(map[string]string{"stage": "error", "message": err.Error()})
		fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		flusher.Flush()
		return
	}
	if !locked {
		errData, _ := json.Marshal(map[string]string{"stage": "error", "message": ErrBookBusy.Message})
		fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		flusher.Flush()
		return
	}

	recoverStatus := model.BookStatusActive
	if book.Status == model.BookStatusOutlining {
		recoverStatus = model.BookStatusOutlining
	}

	output, err := h.pipeline.WriteChapter(c.Request.Context(), pipeline.WriteChapterInput{
		BookID:    uint(id),
		ModelID:   body.ModelID,
		UserInput: body.UserInput,
		Progress:  &sseWriter{w: c.Writer, flusher: flusher},
	})
	if err != nil {
		_ = h.truthRepo.UpdateBookStatus(book.ID, recoverStatus)
		errData, _ := json.Marshal(map[string]string{"stage": "error", "message": err.Error()})
		fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		flusher.Flush()
		return
	}

	_ = h.truthRepo.UpdateBookStatus(book.ID, model.BookStatusActive)

	resultData, _ := json.Marshal(map[string]interface{}{
		"stage":          "complete",
		"chapter_number": output.ChapterNumber,
		"title":          output.Title,
		"content":        output.Content,
		"memo":           output.Memo,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", resultData)
	flusher.Flush()
}

func (h *BookHandler) GetChapter(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	chapterNum, err := strconv.ParseUint(c.Param("chapter"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	chapter, err := h.truthRepo.GetChapter(uint(bookID), uint(chapterNum))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	c.JSON(http.StatusOK, chapter)
}

func (h *BookHandler) DeleteChapter(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	chapterNum, err := strconv.ParseUint(c.Param("chapter"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	if _, err := h.truthRepo.GetChapter(uint(bookID), uint(chapterNum)); err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	if err := h.truthRepo.DeleteLatestChapterCascade(uint(bookID), uint(chapterNum)); err != nil {
		ErrJSON(c, http.StatusBadRequest, ErrChapterDeleteFailed.Code, ErrChapterDeleteFailed.Message+"："+err.Error())
		return
	}

	nextStatus := model.BookStatusActive
	if uint(chapterNum) <= 1 {
		nextStatus = model.BookStatusOutlining
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "章节及其直接关联数据已删除",
		"deleted_chapter": chapterNum,
		"book_status":     nextStatus,
	})
}

func (h *BookHandler) GetTruthFiles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	characters, _ := h.truthRepo.GetCharacters(uint(id))
	facts, _ := h.truthRepo.GetFacts(uint(id))
	hooks, _ := h.truthRepo.GetHooks(uint(id))
	summaries, _ := h.truthRepo.GetChapterSummaries(uint(id))
	snapshots, _ := h.truthRepo.GetChapterSnapshots(uint(id))
	foundations, _ := h.truthRepo.ListFoundations(uint(id))
	bookState, _ := h.truthRepo.GetBookState(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"characters":  characters,
		"facts":       facts,
		"hooks":       hooks,
		"summaries":   summaries,
		"snapshots":   snapshots,
		"foundations": foundations,
		"book_state":  bookState,
	})
}

func (h *BookHandler) GetChapterArtifacts(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	chapterNum, err := strconv.ParseUint(c.Param("chapter"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	userID := c.GetUint("user_id")
	if book.UserID != userID {
		ErrForbidden.JSON(c)
		return
	}

	artifacts, _ := h.truthRepo.GetRuntimeArtifacts(uint(bookID), uint(chapterNum))

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
	})
}

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (s *sseWriter) Write(p []byte) (int, error) {
	n, err := s.w.Write(p)
	return n, err
}

func (s *sseWriter) Flush() error {
	s.flusher.Flush()
	return nil
}
