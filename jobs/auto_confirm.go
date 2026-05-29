package jobs

import (
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"time"

	"github.com/robfig/cron/v3"
)

// StartAutoConfirmJob 启动定时任务，每12小时扫描一次发货超过10天的订单，自动确认收货
func StartAutoConfirmJob() {
	c := cron.New()
	// 每12小时执行一次
	_, err := c.AddFunc("@every 12h", autoConfirmExpiredOrders)
	if err != nil {
		log.Fatalf("添加定时任务失败: %v", err)
	}
	c.Start()
	log.Println("自动确认收货定时任务已启动（每12小时扫描一次）")
}

// autoConfirmExpiredOrders 自动确认发货超过10天但用户未主动确认的订单
func autoConfirmExpiredOrders() {
	tenDaysAgo := time.Now().Add(-10 * 24 * time.Hour)

	var orders []models.Order
	err := database.DB.
		Where("status = ? AND shipped_at IS NOT NULL AND shipped_at <= ?",
			models.OrderStatusShipped, tenDaysAgo).
		Find(&orders).Error
	if err != nil {
		log.Printf("查询待自动确认订单失败: %v", err)
		return
	}

	if len(orders) == 0 {
		return
	}

	now := utils.LocalTime(time.Now())
	for _, order := range orders {
		// 双重检查状态，防止并发修改
		if order.Status != models.OrderStatusShipped {
			continue
		}
		order.Status = models.OrderStatusCompleted
		order.FinishAt = &now
		if err := database.DB.Save(&order).Error; err != nil {
			log.Printf("自动完成订单 %d 失败: %v", order.ID, err)
		} else {
			log.Printf("订单 %s（ID:%d）已自动确认收货（发货超过10天）", order.OrderNo, order.ID)
		}
	}
}
