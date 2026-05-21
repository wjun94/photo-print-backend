package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

// GetUserInfo 获取当前小程序用户信息
// @Summary 获取当前用户信息
// @Tags 小程序用户
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=models.WxUser}
// @Router /api/v1/user/info [get]
func GetUserInfo(c *gin.Context) {
	// 从中间件获取当前用户ID（由 AuthMiddleware 设置）
	// 使用辅助函数获取用户ID（int64）
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录或用户ID无效")
		return
	}

	var user models.WxUser
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	utils.Success(c, user)
}
