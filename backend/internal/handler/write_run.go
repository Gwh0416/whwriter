package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/pipeline"

	"github.com/gin-gonic/gin"
)

func (h *BookHandler) StartWriteRun(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if book.UserID != c.GetUint("user_id") {
		ErrForbidden.JSON(c)
		return
	}

	var req model.StartWriteRunRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil && !errors.Is(bindErr, io.EOF) {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, bindErr.Error())
		return
	}
	if req.ModelID == 0 {
		req.ModelID = book.LLMModelID
	}

	run, err := h.pipeline.StartWriteRun(c.Request.Context(), pipeline.StartWriteRunInput{
		BookID:      uint(bookID),
		ModelID:     req.ModelID,
		UserInput:   req.UserInput,
		RetryMode:   req.RetryMode,
		ParentRunID: nonZeroUintPointer(req.ParentRunID),
	})
	if err != nil {
		if err.Error() == "该书已有写作任务在进行中" {
			ErrBookBusy.JSON(c)
			return
		}
		ErrJSON(c, ErrBookWriteFailed.StatusCode, ErrBookWriteFailed.Code, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"run": run})
}

func (h *BookHandler) ListWriteRuns(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if book.UserID != c.GetUint("user_id") {
		ErrForbidden.JSON(c)
		return
	}
	runs, err := h.truthRepo.ListChapterWriteRuns(uint(bookID), 10)
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, ErrBookWriteFailed.Code, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

func (h *BookHandler) GetActiveWriteRun(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	book, err := h.truthRepo.GetBook(uint(bookID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if book.UserID != c.GetUint("user_id") {
		ErrForbidden.JSON(c)
		return
	}
	run, err := h.truthRepo.GetActiveChapterWriteRun(uint(bookID))
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, ErrBookWriteFailed.Code, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"run": run})
}

func (h *BookHandler) GetWriteRun(c *gin.Context) {
	run, ok := h.getOwnedWriteRun(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"run": run})
}

func (h *BookHandler) GetWriteRunStages(c *gin.Context) {
	run, ok := h.getOwnedWriteRun(c)
	if !ok {
		return
	}
	stages, err := h.truthRepo.GetChapterWriteStages(run.ID)
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, ErrBookWriteFailed.Code, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"run": run, "stages": stages})
}

func (h *BookHandler) CancelWriteRun(c *gin.Context) {
	run, ok := h.getOwnedWriteRun(c)
	if !ok {
		return
	}
	if err := h.pipeline.CancelWriteRun(run.ID); err != nil {
		ErrJSON(c, http.StatusBadRequest, ErrBookWriteFailed.Code, err.Error())
		return
	}
	updated, _ := h.truthRepo.GetChapterWriteRun(run.ID)
	c.JSON(http.StatusOK, gin.H{"run": updated})
}

func (h *BookHandler) RetryWriteRun(c *gin.Context) {
	run, ok := h.getOwnedWriteRun(c)
	if !ok {
		return
	}
	var req model.RetryWriteRunRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil && !errors.Is(bindErr, io.EOF) {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, bindErr.Error())
		return
	}
	newRun, err := h.pipeline.RetryWriteRun(c.Request.Context(), run.ID, req.Mode)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, ErrBookWriteFailed.Code, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"run": newRun})
}

func (h *BookHandler) getOwnedWriteRun(c *gin.Context) (*model.ChapterWriteRun, bool) {
	runID, err := strconv.ParseUint(c.Param("runID"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return nil, false
	}
	run, err := h.truthRepo.GetChapterWriteRun(uint(runID))
	if err != nil || run == nil {
		ErrInvalidID.JSON(c)
		return nil, false
	}
	book, err := h.truthRepo.GetBook(run.BookID)
	if err != nil {
		ErrInvalidID.JSON(c)
		return nil, false
	}
	if book.UserID != c.GetUint("user_id") {
		ErrForbidden.JSON(c)
		return nil, false
	}
	return run, true
}

func nonZeroUintPointer(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}
