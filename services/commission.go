package services

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"time"
)

// CreateCommissionForOrder 订单完成后创建佣金记录（需在订单状态变为 completed 时调用）
func CreateCommissionForOrder(order models.Order) error {
	// 1. 查找下单用户
	var user models.WxUser
	if err := database.DB.First(&user, order.UserID.Int64()).Error; err != nil {
		return nil // 用户不存在，无佣金
	}
	// 2. 检查是否有上级（推广者）
	if user.InviterID.Int64() == 0 {
		return nil // 无上级，不产生佣金
	}
	// 3. 防重（通过订单ID唯一索引）
	var exist models.Commission
	if err := database.DB.Where("order_id = ?", order.ID).First(&exist).Error; err == nil {
		return nil // 已生成过
	}
	// 4. 获取当前生效的返利比例
	var setting models.CommissionSetting
	if err := database.DB.Order("updated_at desc").First(&setting).Error; err != nil {
		setting.Ratio = 10.0 // 默认值
	}
	// 5. 计算佣金
	amount := order.ActualAmount * setting.Ratio / 100
	commission := models.Commission{
		UserID:      user.InviterID,
		FriendID:    order.UserID,
		OrderID:     order.ID,
		OrderAmount: order.ActualAmount,
		Ratio:       setting.Ratio,
		Amount:      amount,
		Status:      models.CommissionStatusPending,
		CreatedAt:   utils.LocalTime(time.Now()),
	}
	return database.DB.Create(&commission).Error
}
