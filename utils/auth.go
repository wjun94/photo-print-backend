package utils

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			Fail(c, "未提供认证令牌")
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			Fail(c, "令牌格式错误")
			c.Abort()
			return
		}
		claims, err := ParseToken(parts[1])
		if err != nil {
			Fail(c, "无效或过期的令牌")
			c.Abort()
			return
		}
		// 存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_type", claims.UserType)
		c.Next()
	}
}
