package api

import (
	"whwriter/backend/internal/api/handler"
	"whwriter/backend/internal/api/middleware"
	"whwriter/backend/internal/config"
	"whwriter/backend/internal/service"
	"whwriter/backend/internal/store"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config, dbStore *store.UserStore, authSvc *service.AuthService, genreStore *store.GenreStore, platformStore *store.PlatformStore, llmStore *store.LLMConfigStore, bookStore *store.BookStore) *gin.Engine {
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

	authHandler := handler.NewAuthHandler(authSvc)
	resourceHandler := handler.NewResourceHandler(genreStore, platformStore, llmStore)
	bookHandler := handler.NewBookHandler(bookStore)

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
			protected.GET("/platforms", resourceHandler.ListPlatforms)
			protected.GET("/llm-configs", resourceHandler.ListLLMConfigs)

			protected.POST("/books", bookHandler.Create)
			protected.GET("/books", bookHandler.List)
		}
	}

	return r
}
