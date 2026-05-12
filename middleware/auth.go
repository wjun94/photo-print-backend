package middleware

import (
	"photo-print-backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Fail(c, "未提供认证令牌")
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Fail(c, "令牌格式错误")
			c.Abort()
			return
		}
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Fail(c, "无效或过期的令牌")
			c.Abort()
			return
		}
		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
