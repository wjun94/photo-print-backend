package controllers

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

// Login 后台登录
// @Summary 后台登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body LoginReq true "用户名密码"
// @Success 200 {object} utils.Response{data=object{token=string}}
// @Router /api/v1/login [post]
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}
	var user models.User
	result := database.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		utils.Fail(c, "用户名或密码错误")
		return
	}
	if !user.CheckPassword(req.Password) {
		utils.Fail(c, "用户名或密码错误")
		return
	}
	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		utils.Fail(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{"token": token})
}
