package jobs

import (
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"time"

	"github.com/robfig/cron/v3"
)

// StartAutoCloseOrderJob 启动定时任务：每60分钟扫描超时未支付订单，自动关闭
func StartAutoCloseOrderJob() {
	c := cron.New()
	// 每 60 分钟执行一次
	_, err := c.AddFunc("*/60 * * * *", autoCloseExpiredUnpaidOrders)
	if err != nil {
		log.Fatalf("添加自动关闭订单定时任务失败: %v", err)
	}
	c.Start()
	log.Println("自动关闭超时未支付订单任务已启动（每分钟扫描一次）")
}

// autoCloseExpiredUnpaidOrders 自动关闭 30 分钟未支付的订单
func autoCloseExpiredUnpaidOrders() {
	// 超时时间：30分钟未支付自动关闭
	expireTime := time.Now().Add(-30 * time.Minute)

	var orders []models.Order
	err := database.DB.
		Where("status = ? AND created_at <= ?", models.OrderStatusPending, expireTime).
		Find(&orders).Error
	if err != nil {
		log.Printf("查询超时未支付订单失败: %v", err)
		return
	}

	if len(orders) == 0 {
		return
	}

	now := utils.LocalTime(time.Now())
	for _, order := range orders {
		// 状态二次校验，防止并发问题
		if order.Status != models.OrderStatusPending {
			continue
		}

		// 关闭订单 + 设置取消时间
		order.Status = models.OrderStatusCancelled
		order.CancelAt = &now

		if err := database.DB.Save(&order).Error; err != nil {
			log.Printf("自动关闭订单 %s 失败: %v", order.OrderNo, err)
		} else {
			log.Printf("订单 %s（ID:%d）已自动关闭（超时未支付）", order.OrderNo, order.ID)
		}
	}
}
