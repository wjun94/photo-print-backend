package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminCompleteOrderReq 管理员手动完成订单请求
type AdminCompleteOrderReq struct {
	OrderID string `json:"orderId" binding:"required"` // 订单ID
}

// AdminCompleteOrder 管理员手动将订单标记为已完成（用于异常处理或售后完结）
// @Summary 管理员手动完成订单
// @Description 将指定订单强制标记为已完成，适用于异常处理或售后完结。仅当订单当前状态不是已完成时有效。
// @Tags 后台-订单管理
// @Accept json
// @Produce json
// @Param request body AdminCompleteOrderReq true "订单ID"
// @Success 200 {object} utils.Response "操作成功"
// @Failure 400 {object} utils.Response "参数错误或订单已是已完成状态"
// @Failure 404 {object} utils.Response "订单不存在"
// @Failure 500 {object} utils.Response "操作失败"
// @Router /api/v1/admin/order/complete [post]
func AdminCompleteOrder(c *gin.Context) {
	var req AdminCompleteOrderReq
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

	if order.Status == models.OrderStatusCompleted {
		utils.Fail(c, "订单已经是已完成状态")
		return
	}

	now := utils.LocalTime(time.Now())
	order.Status = models.OrderStatusCompleted
	order.FinishAt = &now

	if err := database.DB.Save(&order).Error; err != nil {
		utils.Fail(c, "操作失败")
		return
	}

	utils.Success(c, nil)
}
