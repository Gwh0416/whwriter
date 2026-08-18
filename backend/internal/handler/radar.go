package handler

import (
	"net/http"
	"strconv"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type RadarHandler struct {
	radarSvc *service.RadarService
}

func NewRadarHandler(radarSvc *service.RadarService) *RadarHandler {
	return &RadarHandler{radarSvc: radarSvc}
}

func (h *RadarHandler) Overview(c *gin.Context) {
	overview, err := h.radarSvc.Overview()
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, CodeInternal, err.Error())
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *RadarHandler) ListTaxonomies(c *gin.Context) {
	if c.Query("ready_only") == "1" || c.Query("ready_only") == "true" {
		tags, err := h.radarSvc.WritableTags()
		if err != nil {
			ErrJSON(c, http.StatusInternalServerError, CodeInternal, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"taxonomies": []model.RadarTaxonomy{},
			"tags":       tags,
		})
		return
	}
	overview, err := h.radarSvc.Overview()
	if err != nil {
		ErrJSON(c, http.StatusInternalServerError, CodeInternal, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"taxonomies": overview.Taxonomies,
		"tags":       overview.Tags,
	})
}

func (h *RadarHandler) CheckBrowserSession(c *gin.Context) {
	status := h.radarSvc.CheckBrowserSession(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (h *RadarHandler) OpenBrowserLoginPage(c *gin.Context) {
	status := h.radarSvc.OpenBrowserLoginPage(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (h *RadarHandler) CreateSource(c *gin.Context) {
	var req model.CreateRadarSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	source, err := h.radarSvc.CreateManualSource(c.Request.Context(), req)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"source": source})
}

func (h *RadarHandler) AnalyzeSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	var req struct {
		ModelID uint `json:"model_id"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ErrBadRequest.JSON(c)
			return
		}
	}
	profile, err := h.radarSvc.AnalyzeSource(c.Request.Context(), uint(id), req.ModelID)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

func (h *RadarHandler) ListSourceChapters(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	chapters, err := h.radarSvc.ListSourceChapterSamples(uint(id))
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"chapters": chapters})
}

func (h *RadarHandler) CreateScanJob(c *gin.Context) {
	var req model.CreateRadarScanJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	job, err := h.radarSvc.CreateCategoryScanJob(c.Request.Context(), req)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"job": job})
}

func (h *RadarHandler) AnalyzeCategory(c *gin.Context) {
	var req struct {
		Platform string `json:"platform"`
		Category string `json:"category"`
		TagKey   string `json:"tag_key"`
		ModelID  uint   `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	if req.Category == "" {
		req.Category = req.TagKey
	}
	if req.Category == "" {
		ErrBadRequest.JSON(c)
		return
	}
	platform := req.Platform
	if platform == "" {
		platform = model.RadarPlatformFanqie
	}
	count, err := h.radarSvc.AnalyzeCategory(c.Request.Context(), platform, req.Category, req.ModelID)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "单书画像已生成", "count": count})
}

func (h *RadarHandler) ScanIntroSamples(c *gin.Context) {
	var req model.CreateRadarIntroScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	count, err := h.radarSvc.ScanIntroSamples(c.Request.Context(), req)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "简介样本已扫描", "count": count})
}

func (h *RadarHandler) GenerateIntro(c *gin.Context) {
	var req model.GenerateRadarIntroRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	result, err := h.radarSvc.GenerateIntro(c.Request.Context(), req)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

func (h *RadarHandler) DeleteIntroSample(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	count, err := h.radarSvc.DeleteIntroSamples([]uint{uint(id)})
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "简介样本已删除", "count": count})
}

func (h *RadarHandler) DeleteIntroSamples(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	count, err := h.radarSvc.DeleteIntroSamples(req.IDs)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "简介样本已删除", "count": count})
}

func (h *RadarHandler) DeleteScanJob(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.radarSvc.DeleteScanJob(uint(id)); err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "扫描任务已删除"})
}

func (h *RadarHandler) DeleteSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.radarSvc.DeleteSource(uint(id)); err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "书籍样本已删除"})
}

func (h *RadarHandler) DeleteSources(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	count, err := h.radarSvc.DeleteSources(req.IDs)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "书籍样本已删除", "count": count})
}

func (h *RadarHandler) DeleteTaxonomyProfile(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.radarSvc.DeleteTaxonomyProfile(uint(id)); err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "聚合画像已删除"})
}

func (h *RadarHandler) DeleteTaxonomyProfiles(c *gin.Context) {
	var req struct {
		Platform   string   `json:"platform"`
		Categories []string `json:"categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	count, err := h.radarSvc.DeleteTaxonomyProfilesByCategories(req.Platform, req.Categories)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "聚合画像已删除", "count": count})
}

func (h *RadarHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ErrInvalidID.JSON(c)
		return
	}
	if err := h.radarSvc.DeleteRule(uint(id)); err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "写作规则已删除"})
}

func (h *RadarHandler) DeleteRules(c *gin.Context) {
	var req struct {
		Platform   string   `json:"platform"`
		Categories []string `json:"categories"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	count, err := h.radarSvc.DeleteRulesByCategories(req.Platform, req.Categories)
	if err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "写作规则已删除", "count": count})
}

func (h *RadarHandler) Synthesize(c *gin.Context) {
	var req struct {
		Platform string `json:"platform"`
		Category string `json:"category"`
		TagKey   string `json:"tag_key"`
		ModelID  uint   `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrBadRequest.JSON(c)
		return
	}
	if req.Category == "" {
		req.Category = req.TagKey
	}
	if req.Category == "" {
		ErrBadRequest.JSON(c)
		return
	}
	platform := req.Platform
	if platform == "" {
		platform = model.RadarPlatformFanqie
	}
	if err := h.radarSvc.Synthesize(c.Request.Context(), platform, req.Category, req.TagKey, req.ModelID); err != nil {
		ErrJSON(c, http.StatusBadRequest, CodeBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "画像与规则已更新"})
}
