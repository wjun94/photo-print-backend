package jobs

import (
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"time"

	"github.com/robfig/cron/v3"
)

// 注册该定时任务，设置每天执行一次，清理过期超过7天的优惠券
func InitCronJobs() {
	c := cron.New()

	// 每天凌晨 2:00 执行一次清理任务
	_, err := c.AddFunc("0 2 * * *", CleanExpiredCoupons)
	if err != nil {
		panic("定时任务注册失败: " + err.Error())
	}

	c.Start()
	log.Println("定时任务已启动，每天凌晨2点清理过期优惠券")
}

// CleanExpiredCoupons 清理过期超过7天的优惠券（物理删除）
func CleanExpiredCoupons() {
	// 计算 7 天前的时间点
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	// Unscoped() 执行物理删除
	result := database.DB.Unscoped().
		Where("valid_end < ?", sevenDaysAgo).
		Delete(&models.UserCoupon{})

	if result.Error != nil {
		log.Printf("[定时任务] 清理超过7天的过期优惠券失败: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[定时任务] 清理超过7天的过期优惠券成功，共删除 %d 条记录", result.RowsAffected)
	}
}
