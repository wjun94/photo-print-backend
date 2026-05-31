package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PaySuccess 模拟支付成功（实际生产应由微信异步通知调用，此处为开发测试接口）
func PaySuccess(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	orderID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效订单ID")
		return
	}

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}

	// 只有待付款订单可以支付
	if order.Status != models.OrderStatusPending {
		utils.Fail(c, "订单状态不是待付款，无法支付")
		return
	}

	now := utils.LocalTime(time.Now())
	order.Status = models.OrderStatusPaid
	order.PayAt = &now

	if err := database.DB.Save(&order).Error; err != nil {
		utils.Fail(c, "更新订单失败")
		return
	}

	utils.Success(c, nil)
}
