package jobs

import (
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// StartDeleteCancelledOrdersJob 启动删除已取消订单的定时任务（每天凌晨3点执行）
func StartDeleteCancelledOrdersJob() {
	c := cron.New()
	// 每天凌晨3点执行
	_, err := c.AddFunc("0 3 * * *", deleteExpiredCancelledOrders)
	if err != nil {
		log.Fatalf("添加删除已取消订单定时任务失败: %v", err)
	}
	c.Start()
	log.Printf("删除已取消订单定时任务已启动，保留天数: %d", 30)
}

// deleteExpiredCancelledOrders 删除超过保留天数的已取消订单
func deleteExpiredCancelledOrders() {
	retentionDays := 30
	if retentionDays <= 0 {
		log.Println("已取消订单保留天数 <= 0，跳过删除")
		return
	}
	deadline := time.Now().AddDate(0, 0, -retentionDays)

	var orders []models.Order
	err := database.DB.Where("status = ? AND cancel_at IS NOT NULL AND cancel_at <= ?",
		models.OrderStatusCancelled, deadline).Find(&orders).Error
	if err != nil {
		log.Printf("查询待删除取消订单失败: %v", err)
		return
	}
	if len(orders) == 0 {
		return
	}

	// 逐订单删除（事务）
	for _, order := range orders {
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// 删除订单项
			if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderItem{}).Error; err != nil {
				return err
			}
			// 删除地址快照
			if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderAddress{}).Error; err != nil {
				return err
			}
			// 删除订单主表
			if err := tx.Delete(&order).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			log.Printf("删除订单 %s 失败: %v", order.OrderNo, err)
		} else {
			log.Printf("已删除过期取消订单: %s (取消时间: %s)", order.OrderNo, order.CancelAt.String())
		}
	}
}
