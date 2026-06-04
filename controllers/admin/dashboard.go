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
	NewUsersToday  int64             `json:"newUsersToday"`  // 今日新增用户
	TodaySales     float64           `json:"todaySales"`     // 今日销售额
	MonthSales     float64           `json:"monthSales"`     // 本月销售额
	ProductRanking []ProductRankItem `json:"productRanking"` // 商品销售排行
}

// ProductRankItem 商品排行项
type ProductRankItem struct {
	ProductID   string  `json:"productId"`   // 商品ID
	ProductName string  `json:"productName"` // 商品名称
	TotalSales  int     `json:"totalSales"`  // 销售数量
	TotalAmount float64 `json:"totalAmount"` // 销售金额
}

// TrendResponse 趋势数据
type TrendResponse struct {
	Dates  []string  `json:"dates"`  // 日期标签
	Orders []int64   `json:"orders"` // 订单量
	Sales  []float64 `json:"sales"`  // 销售额
}

// TrendRequest 趋势请求参数
type TrendRequest struct {
	Type string `form:"type" example:"day" enums:"day,week,month" description:"维度: day/week/month"` // 维度
}

// GetOverview
// @Summary      获取管理后台概览数据
// @Description  返回今日新增用户、今日/本月销售额、商品销售排行（基于已付款订单）
// @Tags         管理后台
// @Accept       json
// @Produce      json
// @Success      200  {object}  utils.Response{data=OverviewResponse}  "成功"
// @Failure      500  {object}  utils.Response  "服务器内部错误"
// @Router       /admin/overview [get]
func GetOverview(c *gin.Context) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 今日新增用户
	var newUsersToday int64
	database.DB.Model(&models.WxUser{}).Where("created_at >= ?", todayStart).Count(&newUsersToday)

	// 今日销售额（已支付订单，按支付时间）
	var todaySales float64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND pay_at >= ?", models.OrderStatusPaid, todayStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&todaySales)

	// 本月销售额（已支付订单，按支付时间）
	var monthSales float64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND pay_at >= ?", models.OrderStatusPaid, monthStart).
		Select("COALESCE(SUM(actual_amount), 0)").
		Scan(&monthSales)

	// 商品销售排行（按销量，本月已支付订单，按支付时间）
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
		Where("orders.status = ?", models.OrderStatusPaid).
		Where("orders.pay_at >= ? AND orders.pay_at <= ?", monthStart, now).
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

// GetTrend
// @Summary      获取订单量/销售额趋势数据
// @Description  根据维度（日/周/月）返回最近一段时间内的订单量和销售额趋势（基于已付款订单）
// @Tags         管理后台
// @Accept       json
// @Produce      json
// @Param        type  query   string  false  "维度"  default(day)  enums(day,week,month)
// @Success      200  {object}  utils.Response{data=TrendResponse}  "成功"
// @Failure      400  {object}  utils.Response  "参数错误"
// @Failure      500  {object}  utils.Response  "服务器内部错误"
// @Router       /admin/trend [get]
func GetTrend(c *gin.Context) {
	var req TrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

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
		// 最近7天（含今天）
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
			end := start.AddDate(0, 0, 1)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, start, end).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, start, end).
				Count(&orderCnt)
			dateLabels = append(dateLabels, start.Format("01/02"))
			orderCounts = append(orderCounts, orderCnt)
			salesAmounts = append(salesAmounts, sales)
		}
	case "week":
		// 最近4周
		for i := 3; i >= 0; i-- {
			// 计算每周起始（周一）
			weekStart := now.AddDate(0, 0, -int(now.Weekday())-7*i)
			weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
			weekEnd := weekStart.AddDate(0, 0, 7)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, weekStart, weekEnd).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, weekStart, weekEnd).
				Count(&orderCnt)
			dateLabels = append(dateLabels, weekStart.Format("01/02")+"-"+weekEnd.AddDate(0, 0, -1).Format("01/02"))
			orderCounts = append(orderCounts, orderCnt)
			salesAmounts = append(salesAmounts, sales)
		}
	case "month":
		// 最近6个月
		for i := 5; i >= 0; i-- {
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -i, 0)
			monthEnd := monthStart.AddDate(0, 1, 0)
			var orderCnt int64
			var sales float64
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, monthStart, monthEnd).
				Select("COALESCE(SUM(actual_amount), 0)").
				Scan(&sales)
			database.DB.Model(&models.Order{}).
				Where("status = ? AND pay_at >= ? AND pay_at < ?", models.OrderStatusPaid, monthStart, monthEnd).
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
