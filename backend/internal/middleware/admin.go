package middleware

import (
	"net/http"

	"whwriter/backend/internal/model"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != string(model.RoleAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可访问"})
			c.Abort()
			return
		}
		c.Next()
	}
}
