package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BindInviterReq struct {
	InviteCode string `json:"inviteCode" binding:"required"` // 邀请人的用户ID
}

// BindInviter 绑定上级（推广者）
// @Summary 绑定上级（推广者）
// @Description 当前登录用户绑定邀请人作为上级，仅限未绑定过上级的用户使用一次
// @Tags 推广
// @Accept json
// @Produce json
// @Param body body BindInviterReq true "邀请码（上级用户ID）"
// @Success 200 {object} utils.Response
// @Router /api/v1/wx/bind-inviter [post]
func BindInviter(c *gin.Context) {
	// 获取当前登录用户ID
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req BindInviterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	inviterID, err := strconv.ParseInt(req.InviteCode, 10, 64)
	if err != nil {
		utils.Fail(c, "无效的邀请码")
		return
	}

	// 不能绑定自己
	if inviterID == userID {
		utils.Fail(c, "不能绑定自己作为上级")
		return
	}

	// 查询当前用户，检查是否已经绑定过上级
	var user models.WxUser
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.Fail(c, "用户不存在")
		return
	}
	if user.InviterID.Int64() != 0 {
		utils.Fail(c, "您已经绑定过上级，不可重复绑定")
		return
	}

	// 查询上级用户是否存在且未被禁用
	var inviter models.WxUser
	if err := database.DB.First(&inviter, inviterID).Error; err != nil {
		utils.Fail(c, "邀请人不存在")
		return
	}
	if inviter.Status != 1 {
		utils.Fail(c, "邀请人账号异常")
		return
	}

	// 更新当前用户的 inviter_id
	user.InviterID = utils.Int64Str(inviterID)
	if err := database.DB.Save(&user).Error; err != nil {
		utils.Fail(c, "绑定失败，请稍后重试")
		return
	}

	utils.Success(c, nil)
}
