package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// OverviewResponse 概览卡片数据
type OverviewResponse struct {
	NewUsersToday  int64             `json:"newUsersToday"`
	TodaySales     float64           `json:"todaySales"`
	MonthSales     float64           `json:"monthSales"`
	ProductRanking []ProductRankItem `json:"productRanking"`
}

type ProductRankItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	TotalSales  int     `json:"totalSales"`
	TotalAmount float64 `json:"totalAmount"`
}

type TrendResponse struct {
	Dates  []string  `json:"dates"`
	Orders []int64   `json:"orders"`
	Sales  []float64 `json:"sales"`
}

type TrendRequest struct {
	Type string `form:"type" binding:"omitempty,oneof=day week month"`
}

// GetOverview 获取后台概览数据
// @Summary 获取后台概览数据
// @Tags 管理后台
// @Produce json
// @Success 200 {object} utils.Response{data=OverviewResponse}
// @Router /api/v1/admin/dashboard/overview [get]
func GetOverview(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 今日新增用户
	var newUsersToday int64
	database.DB.Model(&models.WxUser{}).Where("created_at >= ?", todayStart).Count(&newUsersToday)

	// 今日销售额（已完成订单，按完成时间）
	var todaySales float64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND finish_at >= ?", models.OrderStatusCompleted, todayStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&todaySales)

	// 本月销售额（已完成订单）
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
		Joins("LEFT JOIN specs ON order_items.spec_id = specs.id").
		Joins("LEFT JOIN products ON specs.product_id = products.id").
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
	ranking := make([]ProductRankItem, 0, len(ranks))
	for _, r := range ranks {
		ranking = append(ranking, ProductRankItem{
			ProductID:   r.ProductID,
			ProductName: r.ProductName,
			TotalSales:  r.TotalQty,
			TotalAmount: r.TotalAmount,
		})
	}

	resp := OverviewResponse{
		NewUsersToday:  newUsersToday,
		TodaySales:     todaySales,
		MonthSales:     monthSales,
		ProductRanking: ranking,
	}
	utils.Success(c, resp)
}

// GetTrend 获取销售趋势数据
// @Summary 获取销售趋势
// @Tags 管理后台
// @Produce json
// @Param type query string false "维度" default(day) enums(day,week,month)
// @Success 200 {object} utils.Response{data=TrendResponse}
// @Router /api/v1/admin/dashboard/trend [get]
func GetTrend(c *gin.Context) {
	var req TrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}
	if req.Type == "" {
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

	resp := TrendResponse{
		Dates:  dateLabels,
		Orders: orderCounts,
		Sales:  salesAmounts,
	}
	utils.Success(c, resp)
}
