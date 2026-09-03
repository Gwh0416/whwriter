package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/pipeline"
	"whwriter/backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type BookHandler struct {
	bookRepo  repository.BookRepository
	truthRepo repository.TruthFileRepository
	radarRepo repository.RadarRepository
	pipeline  *pipeline.Pipeline
}

func NewBookHandler(bookRepo repository.BookRepository, truthRepo repository.TruthFileRepository, pl *pipeline.Pipeline, radarRepo ...repository.RadarRepository) *BookHandler {
	var rr repository.RadarRepository
	if len(radarRepo) > 0 {
		rr = radarRepo[0]
	}
	return &BookHandler{bookRepo: bookRepo, truthRepo: truthRepo, radarRepo: rr, pipeline: pl}
}

func (h *BookHandler) Create(c *gin.Context) {
	var req model.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrIncompleteBook.JSON(c)
		return
	}

	if req.ChapterWordCount <= 0 || req.TargetChapters <= 0 {
		ErrJSON(c, http.StatusBadRequest, CodeIncompleteBook, "每章字数与目标章数必须为正数")
		return
	}
	if len(req.RadarTags) == 0 {
		ErrJSON(c, http.StatusBadRequest, CodeIncompleteBook, "请选择至少一个番茄官方标签")
		return
	}

	book := &model.Book{
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

	if h.radarRepo != nil {
		tagsJSON := "[]"
		if payload, err := json.Marshal(req.RadarTags); err == nil {
			tagsJSON = string(payload)
		}
		primaryTag := strings.TrimSpace(req.RadarCategory)
		if primaryTag == "" && len(req.RadarTags) > 0 {
			primaryTag = strings.TrimSpace(req.RadarTags[0])
		}
		_ = h.radarRepo.SaveBookSetting(&model.RadarBookSetting{
			BookID:   book.ID,
			Platform: model.RadarPlatformFanqie,
			Category: primaryTag,
			TagsJSON: tagsJSON,
		})
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
	books, err := h.bookRepo.List()
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

	if _, err := h.truthRepo.GetBook(uint(id)); err != nil {
		ErrInvalidID.JSON(c)
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

	if _, err := h.truthRepo.GetBook(uint(bookID)); err != nil {
		ErrInvalidID.JSON(c)
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

	if _, err := h.truthRepo.GetBook(uint(bookID)); err != nil {
		ErrInvalidID.JSON(c)
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

	if _, err := h.truthRepo.GetBook(uint(id)); err != nil {
		ErrInvalidID.JSON(c)
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

func (h *BookHandler) ListWikiEntities(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if _, err := h.truthRepo.GetBook(uint(bookID)); err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	limit := parsePositiveQueryInt(c.Query("limit"), 100, 500)
	offset := parsePositiveQueryInt(c.Query("offset"), 0, 100000)
	entities, total, err := h.truthRepo.SearchWikiEntities(
		uint(bookID),
		strings.TrimSpace(c.Query("q")),
		parseWikiEntityTypes(c.Query("type")),
		limit,
		offset,
	)
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, CodeInternal, "加载 Wiki 实体失败："+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":  entities,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *BookHandler) GetWikiEntity(c *gin.Context) {
	bookID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	entityID, err := strconv.ParseUint(c.Param("entityID"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if _, err := h.truthRepo.GetBook(uint(bookID)); err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	page, err := h.truthRepo.GetWikiEntityPage(uint(bookID), uint(entityID))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	c.JSON(http.StatusOK, page)
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

	if _, err := h.truthRepo.GetBook(uint(bookID)); err != nil {
		ErrInvalidID.JSON(c)
		return
	}

	artifacts, _ := h.truthRepo.GetRuntimeArtifacts(uint(bookID), uint(chapterNum))

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
	})
}

func parsePositiveQueryInt(raw string, fallback, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return fallback
	}
	if maximum > 0 && value > maximum {
		return maximum
	}
	return value
}

func parseWikiEntityTypes(raw string) []model.WikiEntityType {
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '|'
	})
	types := make([]model.WikiEntityType, 0, len(values))
	for _, value := range values {
		switch entityType := model.WikiEntityType(strings.TrimSpace(value)); entityType {
		case model.WikiEntityCharacter,
			model.WikiEntityPlace,
			model.WikiEntityItem,
			model.WikiEntityOrganization,
			model.WikiEntityEvent,
			model.WikiEntityHook,
			model.WikiEntityRule,
			model.WikiEntityConcept:
			types = append(types, entityType)
		}
	}
	return types
}

func (h *BookHandler) ExportBook(c *gin.Context) {
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

	format := c.DefaultQuery("format", "txt")
	if format != "txt" && format != "md" {
		format = "txt"
	}

	chapters, err := h.truthRepo.ListChapters(uint(id))
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if len(chapters) == 0 {
		ErrJSON(c, http.StatusBadRequest, CodeIncompleteBook, "该书暂无章节，无法导出")
		return
	}

	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].ChapterNumber < chapters[j].ChapterNumber
	})

	var builder strings.Builder
	if format == "md" {
		builder.WriteString("# " + book.Title + "\n\n")
		for _, ch := range chapters {
			title := ch.Title
			if title == "" {
				title = "未命名"
			}
			fmt.Fprintf(&builder, "## 第%d章 %s\n\n%s\n\n", ch.ChapterNumber, title, ch.Content)
		}
	} else {
		builder.WriteString(book.Title + "\n\n")
		for _, ch := range chapters {
			title := ch.Title
			if title == "" {
				title = "未命名"
			}
			fmt.Fprintf(&builder, "第%d章 %s\n\n%s\n\n", ch.ChapterNumber, title, ch.Content)
		}
	}

	ext := "txt"
	ctype := "text/plain; charset=utf-8"
	if format == "md" {
		ext = "md"
		ctype = "text/markdown; charset=utf-8"
	}
	filename := book.Title + "." + ext
	c.Header("Content-Type", ctype)
	c.Header("Content-Disposition", "attachment; filename=\"export."+ext+"\"; filename*=UTF-8''"+url.PathEscape(filename))
	c.String(http.StatusOK, builder.String())
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
