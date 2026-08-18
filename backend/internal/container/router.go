package container

import (
	"whwriter/backend/internal/handler"
	"whwriter/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(c *Container) *gin.Engine {
	if !c.Config.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.GET("/health", handler.Health)

	r.Static("/assets", "../frontend/dist/assets")
	r.StaticFile("/favicon.ico", "../frontend/dist/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		c.File("../frontend/dist/index.html")
	})
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "GET" && c.NegotiateFormat(gin.MIMEHTML) == gin.MIMEHTML {
			c.File("../frontend/dist/index.html")
		} else {
			c.JSON(404, gin.H{"error": "not found"})
		}
	})

	resourceHandler := handler.NewResourceHandler(c.GenreRepo, c.PlatformRepo, c.LLMModelRepo, c.LLMConfigRepo)
	bookHandler := handler.NewBookHandler(c.BookRepo, c.TruthFileRepo, c.Pipeline, c.RadarRepo)
	radarHandler := handler.NewRadarHandler(c.RadarSvc)
	settingsHandler := handler.NewSettingsHandler(c.GenreRepo, c.PlatformRepo, c.LLMConfigRepo, c.LLMModelRepo, c.DB)

	api := r.Group("/api/v1")
	{
		api.GET("/ping", handler.Ping)

		{
			api.POST("/initialize", settingsHandler.Initialize)
			api.GET("/stats", settingsHandler.GetStats)

			api.GET("/genres", resourceHandler.ListGenres)
			api.POST("/genres", settingsHandler.CreateGenre)
			api.PUT("/genres/:id", settingsHandler.UpdateGenre)
			api.DELETE("/genres/:id", settingsHandler.DeleteGenre)

			api.GET("/platforms", resourceHandler.ListPlatforms)
			api.POST("/platforms", settingsHandler.CreatePlatform)
			api.PUT("/platforms/:id", settingsHandler.UpdatePlatform)
			api.DELETE("/platforms/:id", settingsHandler.DeletePlatform)

			api.GET("/llm-configs", settingsHandler.ListLLMConfigs)
			api.GET("/llm-configs/token-usage", settingsHandler.GetTokenUsageStats)
			api.POST("/llm-configs/test-connection", settingsHandler.TestLLMConnection)
			api.POST("/llm-configs", settingsHandler.CreateLLMConfig)
			api.PUT("/llm-configs/:id", settingsHandler.UpdateLLMConfig)
			api.DELETE("/llm-configs/:id", settingsHandler.DeleteLLMConfig)
			api.POST("/llm-configs/:id/models", settingsHandler.SaveLLMModels)
			api.POST("/llm-models/:id/default", settingsHandler.SetDefaultModel)

			api.GET("/radar/overview", radarHandler.Overview)
			api.GET("/radar/taxonomies", radarHandler.ListTaxonomies)
			api.GET("/radar/browser/check", radarHandler.CheckBrowserSession)
			api.POST("/radar/browser/open", radarHandler.OpenBrowserLoginPage)
			api.POST("/radar/sources", radarHandler.CreateSource)
			api.POST("/radar/sources/delete", radarHandler.DeleteSources)
			api.DELETE("/radar/sources/:id", radarHandler.DeleteSource)
			api.POST("/radar/sources/:id/analyze", radarHandler.AnalyzeSource)
			api.GET("/radar/sources/:id/chapters", radarHandler.ListSourceChapters)
			api.POST("/radar/categories/analyze", radarHandler.AnalyzeCategory)
			api.POST("/radar/intros/scan", radarHandler.ScanIntroSamples)
			api.POST("/radar/intros/generate", radarHandler.GenerateIntro)
			api.POST("/radar/intros/delete", radarHandler.DeleteIntroSamples)
			api.DELETE("/radar/intros/:id", radarHandler.DeleteIntroSample)
			api.POST("/radar/scan-jobs", radarHandler.CreateScanJob)
			api.DELETE("/radar/scan-jobs/:id", radarHandler.DeleteScanJob)
			api.POST("/radar/profiles/delete", radarHandler.DeleteTaxonomyProfiles)
			api.DELETE("/radar/profiles/:id", radarHandler.DeleteTaxonomyProfile)
			api.POST("/radar/rules/delete", radarHandler.DeleteRules)
			api.DELETE("/radar/rules/:id", radarHandler.DeleteRule)
			api.POST("/radar/synthesize", radarHandler.Synthesize)

			api.POST("/books", bookHandler.Create)
			api.GET("/books", bookHandler.List)
			api.GET("/books/:id", bookHandler.Get)
			api.DELETE("/books/:id", bookHandler.Delete)
			api.POST("/books/:id/write", bookHandler.WriteChapter)
			api.POST("/books/:id/write-runs", bookHandler.StartWriteRun)
			api.GET("/books/:id/write-runs", bookHandler.ListWriteRuns)
			api.GET("/books/:id/write-runs/active", bookHandler.GetActiveWriteRun)
			api.GET("/books/:id/chapters/:chapter", bookHandler.GetChapter)
			api.DELETE("/books/:id/chapters/:chapter", bookHandler.DeleteChapter)
			api.GET("/books/:id/truth-files", bookHandler.GetTruthFiles)
			api.GET("/books/:id/export", bookHandler.ExportBook)
			api.GET("/books/:id/chapters/:chapter/artifacts", bookHandler.GetChapterArtifacts)
			api.GET("/write-runs/:runID", bookHandler.GetWriteRun)
			api.GET("/write-runs/:runID/stages", bookHandler.GetWriteRunStages)
			api.POST("/write-runs/:runID/cancel", bookHandler.CancelWriteRun)
			api.POST("/write-runs/:runID/retry", bookHandler.RetryWriteRun)
		}
	}

	return r
}
