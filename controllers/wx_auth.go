package controllers

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

// struct 小程序登录
// @Summary 小程序登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body LoginReq true "生成令牌失败"
// @Success 200 {object} utils.Response{data=object{token=string}}
// @Router /api/v1/wx/login [post]
func WxLogin(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "缺少code")
		return
	}
	// 调用微信接口获取 openid（代码省略，同上文）
	openid := "从微信获取的openid"
	// 查找或创建 WxUser
	var user models.WxUser
	result := database.DB.Where("open_id = ?", openid).First(&user)
	if result.Error != nil {
		user = models.WxUser{OpenID: openid}
		database.DB.Create(&user)
	}
	token, err := utils.GenerateToken(user.ID.Int64(), "", "wx")
	if err != nil {
		utils.Fail(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{"token": token, "user_id": user.ID})
}
