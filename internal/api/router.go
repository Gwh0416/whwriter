package api

import (
	"whwriter/internal/api/handler"
	"whwriter/internal/api/middleware"
	"whwriter/internal/config"
	"whwriter/internal/service"
	"whwriter/internal/store"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config, dbStore *store.UserStore, authSvc *service.AuthService) *gin.Engine {
	if !cfg.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	r.GET("/health", handler.Health)

	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "GET" && c.NegotiateFormat(gin.MIMEHTML) == gin.MIMEHTML {
			c.File("./web/dist/index.html")
		} else {
			c.JSON(404, gin.H{"error": "not found"})
		}
	})

	authHandler := handler.NewAuthHandler(authSvc)

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
		}
	}

	return r
}
