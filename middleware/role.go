package middleware

import (
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists || userType != role {
			utils.Fail(c, "无权限访问")
			c.Abort()
			return
		}
		c.Next()
	}
}
