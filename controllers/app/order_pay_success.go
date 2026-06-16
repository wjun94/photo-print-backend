package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PaySuccessReq 模拟支付成功请求
type PaySuccessReq struct {
	OrderID string `json:"orderId" binding:"required"` // 订单ID
}

// PaySuccess 模拟支付成功（实际生产应由微信异步通知调用，此处为开发测试接口）
// @Summary 模拟支付成功（测试用）
// @Description 将指定订单状态从待付款改为已支付，仅供开发测试使用。实际生产环境应由微信支付异步通知触发。
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body PaySuccessReq true "订单ID"
// @Success 200 {object} utils.Response "支付成功，订单状态已更新"
// @Failure 400 {object} utils.Response "参数错误或订单状态不是待付款"
// @Failure 404 {object} utils.Response "订单不存在"
// @Failure 500 {object} utils.Response "更新订单失败"
// @Router /api/v1/wx/order/pay/success [post]
func PaySuccess(c *gin.Context) {
	var req PaySuccessReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	orderID, err := strconv.ParseInt(req.OrderID, 10, 64)
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
