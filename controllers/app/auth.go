package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// WxLoginReq 小程序登录请求参数
type WxLoginReq struct {
	Code       string `json:"code" binding:"required"`
	InviteCode string `json:"inviteCode"` // 可选，邀请人的用户ID（上级）
}

// WxLogin 小程序静默登录（支持绑定邀请码）
// @Summary 小程序静默登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body WxLoginReq true "登录请求（含inviteCode）"
// @Success 200 {object} utils.Response{data=object{token=string,userId=string}}
// @Router /api/v1/wx/login [post]
func WxLogin(c *gin.Context) {
	var req WxLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "缺少code")
		return
	}

	// TODO: 调用微信接口获取 openid（此处为示例）
	openid := "从微信获取的openid"

	// 查找或创建 WxUser
	var user models.WxUser
	result := database.DB.Where("open_id = ?", openid).First(&user)
	isNewUser := result.Error != nil

	if isNewUser {
		// 新用户：创建记录，并处理邀请码
		user = models.WxUser{OpenID: openid}
		// 处理邀请码（仅新用户有效）
		if req.InviteCode != "" {
			inviterID, err := strconv.ParseInt(req.InviteCode, 10, 64)
			if err == nil && inviterID != user.ID.Int64() {
				// 检查邀请人是否存在且未被禁用
				var inviter models.WxUser
				if database.DB.First(&inviter, inviterID).Error == nil && inviter.Status == 1 {
					user.InviterID = utils.Int64Str(inviterID)
				}
			}
		}
		database.DB.Create(&user)
	} else {
		// 老用户：忽略邀请码（不更新上级）
		// 可选：如果业务允许，也可以更新上级，但通常只在首次绑定时设置
	}

	token, err := utils.GenerateToken(user.ID.Int64(), "", "wx")
	if err != nil {
		utils.Fail(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{
		"token":  token,
		"userId": user.ID.String(),
	})
}
