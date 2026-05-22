package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// struct 后台登录
// @Summary 后台登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body LoginReq true "用户名密码"
// @Success 200 {object} utils.Response{data=object{token=string}}
// @Router /api/v1/login [post]
func AdminLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}
	var admin models.Admin
	result := database.DB.Where("username = ?", req.Username).First(&admin)
	if result.Error != nil || !admin.CheckPassword(req.Password) {
		utils.Fail(c, "用户名或密码错误")
		return
	}
	token, err := utils.GenerateToken(admin.ID.Int64(), admin.Username, "admin")
	if err != nil {
		utils.Fail(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{"token": token})
}

// GetAdminInfo 获取当前登录的管理员信息
// @Summary 获取管理员信息
// @Tags 后台管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=models.Admin}
// @Router /api/v1/admin/info [get]
func GetAdminInfo(c *gin.Context) {
	// 从中间件获取管理员ID（int64）
	adminID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录或用户ID无效")
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, adminID).Error; err != nil {
		utils.Fail(c, "管理员不存在")
		return
	}
	// 不返回密码字段（JSON 标签已有 '-'）
	utils.Success(c, admin)
}
