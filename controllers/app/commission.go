package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTotalCommission 获取累计佣金
// @Summary 获取累计佣金总额
// @Description 推广者查看自己的累计佣金（含待结算和已结算）
// @Tags 佣金
// @Produce json
// @Success 200 {object} utils.Response{data=object{totalCommission=float64}}
// @Router /api/v1/wx/commission/total [get]
func GetTotalCommission(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var total float64
	database.DB.Model(&models.Commission{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total)
	utils.Success(c, gin.H{"totalCommission": total})
}

// GetCommissionList 佣金明细列表（分页）
// @Summary 佣金明细列表
// @Description 推广者查看每笔佣金的详情，支持分页
// @Tags 佣金
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=object{list=[]models.Commission,total=int64,page=int,size=int}}
// @Router /api/v1/wx/commission/list [get]
func GetCommissionList(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size

	var records []models.Commission
	var total int64
	query := database.DB.Model(&models.Commission{}).Where("user_id = ?", userID)
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&records)

	utils.Success(c, gin.H{
		"list":  records,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetInvitedFriends 获取邀请的好友列表
// @Summary 获取邀请的好友列表
// @Description 推广者查看自己直接邀请的下级用户，包含他们的累计消费金额
// @Tags 佣金
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=object{list=[]object{id=string,nickname=string,avatarUrl=string,createdAt=string,totalSpent=float64}}}
// @Router /api/v1/wx/commission/friends [get]
func GetInvitedFriends(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size

	type FriendInfo struct {
		ID         string  `json:"id"`
		Nickname   string  `json:"nickname"`
		AvatarURL  string  `json:"avatarUrl"`
		CreatedAt  string  `json:"createdAt"`
		TotalSpent float64 `json:"totalSpent"`
	}
	var friends []FriendInfo
	err := database.DB.Table("wx_users").
		Select(`wx_users.id, wx_users.nickname, wx_users.avatar_url, wx_users.created_at, 
                COALESCE(SUM(orders.actual_amount), 0) as total_spent`).
		Joins("LEFT JOIN orders ON orders.user_id = wx_users.id AND orders.status = ?", models.OrderStatusCompleted).
		Where("wx_users.inviter_id = ?", userID).
		Group("wx_users.id, wx_users.nickname, wx_users.avatar_url, wx_users.created_at").
		Order("wx_users.created_at desc").
		Offset(offset).Limit(size).
		Scan(&friends).Error
	if err != nil {
		utils.Fail(c, "查询失败")
		return
	}
	utils.Success(c, gin.H{"list": friends})
}
