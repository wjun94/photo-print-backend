package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/models/dto"
	"photo-print-backend/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOverview 获取概览卡片数据
func GetOverview(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 今日新增用户
	var newUsersToday int64
	database.DB.Model(&models.WxUser{}).Where("created_at >= ?", todayStart).Count(&newUsersToday)

	// 今日销售额（已完成订单）
	var todaySales float64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND finish_at >= ?", models.OrderStatusCompleted, todayStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&todaySales)

	// 本月销售额
	var monthSales float64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND finish_at >= ?", models.OrderStatusCompleted, monthStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&monthSales)

	// 商品销售排行（按销量，本月已完成订单）
	type ProductRank struct {
		ProductID   string
		ProductName string
		TotalQty    int
		TotalAmount float64
	}
	var ranks []ProductRank
	err := database.DB.Table("order_items").
		Select(`products.id as product_id, 
                products.name as product_name, 
                SUM(order_items.quantity) as total_qty,
                SUM(order_items.quantity * order_items.price) as total_amount`).
		Joins("LEFT JOIN product_specs ON order_items.spec_id = product_specs.id").
		Joins("LEFT JOIN products ON product_specs.product_id = products.id").
		Joins("INNER JOIN orders ON order_items.order_id = orders.id").
		Where("orders.status = ?", models.OrderStatusCompleted).
		Where("orders.finish_at >= ? AND orders.finish_at <= ?", monthStart, now).
		Group("products.id, products.name").
		Order("total_qty DESC").
		Limit(10).
		Scan(&ranks).Error
	if err != nil {
		utils.Fail(c, "获取商品排行失败")
		return
	}
	ranking := make([]dto.ProductRankItem, 0, len(ranks))
	for _, r := range ranks {
		ranking = append(ranking, dto.ProductRankItem{
			ProductID:   r.ProductID,
			ProductName: r.ProductName,
			TotalSales:  r.TotalQty,
			TotalAmount: r.TotalAmount,
		})
	}

	resp := dto.OverviewResponse{
		NewUsersToday:  newUsersToday,
		TodaySales:     todaySales,
		MonthSales:     monthSales,
		ProductRanking: ranking,
	}
	utils.Success(c, resp)
}

// GetTrend 获取订单量/销售额趋势数据
func GetTrend(c *gin.Context) {
	var req dto.TrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 默认 day，若传入非法值也默认 day
	validTypes := map[string]bool{"day": true, "week": true, "month": true}
	if req.Type == "" || !validTypes[req.Type] {
		req.Type = "day"
	}

	var dateLabels []string
	var orderCounts []int64
	var salesAmounts []float64

	now := time.Now()
	switch req.Type {
	case "day":
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			end := start.AddDate(0, 0, 1)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, start, end).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, start, end).
				Count(&orderCnt)
			dateLabels = append(dateLabels, start.Format("01/02"))
			orderCounts = append(orderCounts, orderCnt)
			salesAmounts = append(salesAmounts, sales)
		}
	case "week":
		for i := 3; i >= 0; i-- {
			weekStart := now.AddDate(0, 0, -int(now.Weekday())-7*i)
			weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
			weekEnd := weekStart.AddDate(0, 0, 7)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, weekStart, weekEnd).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, weekStart, weekEnd).
				Count(&orderCnt)
			dateLabels = append(dateLabels, weekStart.Format("01/02")+"-"+weekEnd.AddDate(0, 0, -1).Format("01/02"))
			orderCounts = append(orderCounts, orderCnt)
			salesAmounts = append(salesAmounts, sales)
		}
	case "month":
		for i := 5; i >= 0; i-- {
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -i, 0)
			monthEnd := monthStart.AddDate(0, 1, 0)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, monthStart, monthEnd).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND finish_at >= ? AND finish_at < ?", models.OrderStatusCompleted, monthStart, monthEnd).
				Count(&orderCnt)
			dateLabels = append(dateLabels, monthStart.Format("2006-01"))
			orderCounts = append(orderCounts, orderCnt)
			salesAmounts = append(salesAmounts, sales)
		}
	}

	resp := dto.TrendResponse{
		Dates:  dateLabels,
		Orders: orderCounts,
		Sales:  salesAmounts,
	}
	utils.Success(c, resp)
}
