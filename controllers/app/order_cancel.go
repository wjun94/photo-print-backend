package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CancelOrderReq 取消订单请求参数
type CancelOrderReq struct {
	OrderID string `json:"orderId" binding:"required"` // 订单ID
}

// CancelOrder 用户取消订单（仅限待付款或已支付未发货状态，取消后自动恢复库存）
// @Summary 取消订单
// @Description 用户取消订单，仅限待付款或已支付未发货状态，取消后自动恢复库存
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body CancelOrderReq true "取消订单请求"
// @Success 200 {object} utils.Response "取消成功"
// @Failure 400 {object} utils.Response "参数错误或状态不允许取消"
// @Failure 404 {object} utils.Response "订单不存在"
// @Router /api/v1/wx/order/cancel [post]
func CancelOrder(c *gin.Context) {
	var req CancelOrderReq
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
	if err := database.DB.Preload("Items").First(&order, orderID).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}

	// 仅待付款或已支付但未发货的订单可取消（已发货不可取消）
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusPaid {
		utils.Fail(c, "当前状态无法取消订单")
		return
	}

	// 事务：更新订单状态为已取消，恢复商品库存
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		now := utils.LocalTime(time.Now())
		order.Status = models.OrderStatusCancelled
		order.CancelAt = &now
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		// 恢复库存（需要 OrderItem 中存储了 SpecID）
		for _, item := range order.Items {
			if err := tx.Model(&models.Spec{}).
				Where("id = ?", item.SpecID).
				Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "取消失败，请重试")
		return
	}

	utils.Success(c, nil)
}
