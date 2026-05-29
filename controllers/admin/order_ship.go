package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShipOrderReq struct {
	OrderID     string `json:"orderId" binding:"required"`
	TrackingNo  string `json:"trackingNo" binding:"required"`
	CourierCode string `json:"courierCode" binding:"required"`
	CourierName string `json:"courierName" binding:"required"`
}

// ShipOrder 管理员发货（支持分批发货，每次创建一条物流记录）
func ShipOrder(c *gin.Context) {
	var req ShipOrderReq
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

	// 只有已支付或已发货的订单才能添加物流（已发货允许再增加包裹）
	if order.Status != models.OrderStatusPaid && order.Status != models.OrderStatusShipped {
		utils.Fail(c, "只有已支付或已发货的订单才能发货")
		return
	}

	// 创建物流记录
	logistics := models.Logistics{
		ID:          utils.Int64Str(utils.NextID()),
		OrderID:     order.ID,
		TrackingNo:  req.TrackingNo,
		CourierCode: req.CourierCode,
		CourierName: req.CourierName,
		CreatedAt:   utils.LocalTime(time.Now()),
		UpdatedAt:   utils.LocalTime(time.Now()),
	}

	now := utils.LocalTime(time.Now())
	needUpdateOrder := false
	if order.Status == models.OrderStatusPaid {
		// 首次发货，更新订单状态为已发货并记录发货时间
		order.Status = models.OrderStatusShipped
		order.ShippedAt = &now
		needUpdateOrder = true
	}

	// 事务：保存物流记录，若首次发货则更新订单
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&logistics).Error; err != nil {
			return err
		}
		if needUpdateOrder {
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "发货失败")
		return
	}

	utils.Success(c, nil)
}
