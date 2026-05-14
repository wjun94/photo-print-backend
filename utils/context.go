package utils

import (
	"github.com/gin-gonic/gin"
)

// GetUserID 从上下文中获取当前登录用户ID（int64）
// 返回值：userID, ok。若 ok == false 表示未登录或用户ID无效
func GetUserID(c *gin.Context) (int64, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	// 尝试多种类型转换
	switch v := val.(type) {
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint64:
		return int64(v), true
	case int:
		return int64(v), true
	case float64:
		// 雪花ID可能作为float64从JSON解析（罕见）
		return int64(v), true
	default:
		return 0, false
	}
}
