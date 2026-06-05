package container

import (
	"whwriter/backend/internal/handler"
	"whwriter/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(c *Container) *gin.Engine {
	cfg := c.Config

	if !cfg.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.Logger())
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

	authHandler := handler.NewAuthHandler(c.AuthSvc)
	resourceHandler := handler.NewResourceHandler(c.GenreRepo, c.PlatformRepo, c.LLMModelRepo, c.LLMConfigRepo)
	bookHandler := handler.NewBookHandler(c.BookRepo, c.TruthFileRepo, c.Pipeline)
	adminHandler := handler.NewAdminHandler(c.UserRepo, c.GenreRepo, c.PlatformRepo, c.LLMConfigRepo, c.LLMModelRepo, c.DB)

	api := r.Group("/api/v1")
	{
		api.GET("/ping", handler.Ping)

		auth := api.Group("/auth")
		{
			auth.POST("/send-code", authHandler.SendCode)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := api.Group("")
		protected.Use(middleware.Auth(cfg))
		{
			protected.GET("/me", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"user_id":  c.GetUint("user_id"),
					"email":    c.GetString("email"),
					"username": c.GetString("username"),
					"role":     c.GetString("role"),
				})
			})

			protected.POST("/auth/send-change-password-code", authHandler.SendChangePasswordCode)
			protected.POST("/auth/change-password", authHandler.ChangePassword)

			protected.GET("/genres", resourceHandler.ListGenres)
			protected.GET("/my-genres", resourceHandler.ListMyGenres)
			protected.POST("/my-genres", resourceHandler.CreateMyGenre)
			protected.PUT("/my-genres/:id", resourceHandler.UpdateMyGenre)
			protected.DELETE("/my-genres/:id", resourceHandler.DeleteMyGenre)
			protected.GET("/platforms", resourceHandler.ListPlatforms)
			protected.GET("/llm-configs", resourceHandler.ListLLMConfigs)

			protected.POST("/books", bookHandler.Create)
			protected.GET("/books", bookHandler.List)
			protected.GET("/books/:id", bookHandler.Get)
			protected.DELETE("/books/:id", bookHandler.Delete)
			protected.POST("/books/:id/write", bookHandler.WriteChapter)
			protected.GET("/books/:id/chapters/:chapter", bookHandler.GetChapter)
			protected.DELETE("/books/:id/chapters/:chapter", bookHandler.DeleteChapter)
			protected.GET("/books/:id/truth-files", bookHandler.GetTruthFiles)
			protected.GET("/books/:id/chapters/:chapter/artifacts", bookHandler.GetChapterArtifacts)

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				admin.GET("/stats", adminHandler.GetStats)
				admin.GET("/users", adminHandler.ListUsers)
				admin.POST("/initialize", adminHandler.Initialize)

				admin.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
				admin.POST("/users/:id/balance", adminHandler.AddBalance)

				admin.GET("/genres", adminHandler.ListGenres)
				admin.POST("/genres", adminHandler.CreateGenre)
				admin.PUT("/genres/:id", adminHandler.UpdateGenre)
				admin.DELETE("/genres/:id", adminHandler.DeleteGenre)

				admin.GET("/platforms", adminHandler.ListPlatforms)
				admin.POST("/platforms", adminHandler.CreatePlatform)
				admin.PUT("/platforms/:id", adminHandler.UpdatePlatform)
				admin.DELETE("/platforms/:id", adminHandler.DeletePlatform)

				admin.GET("/llm-configs", adminHandler.ListLLMConfigs)
				admin.GET("/llm-configs/token-usage", adminHandler.GetTokenUsageStats)
				admin.POST("/llm-configs/test-connection", adminHandler.TestLLMConnection)
				admin.POST("/llm-configs", adminHandler.CreateLLMConfig)
				admin.PUT("/llm-configs/:id", adminHandler.UpdateLLMConfig)
				admin.DELETE("/llm-configs/:id", adminHandler.DeleteLLMConfig)
				admin.POST("/llm-configs/:id/models", adminHandler.SaveLLMModels)
				admin.POST("/llm-models/:id/default", adminHandler.SetDefaultModel)
			}
		}
	}

	return r
}
