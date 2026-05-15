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
			utils.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Unauthorized(c, "令牌格式错误")
			c.Abort()
			return
		}
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Unauthorized(c, "无效或过期的令牌")
			c.Abort()
			return
		}
		// 验证 user_id 有效（非零）
		if claims.UserID == 0 {
			utils.Unauthorized(c, "用户ID无效")
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
