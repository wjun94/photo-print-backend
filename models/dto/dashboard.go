package dto

// OverviewResponse 概览卡片数据
type OverviewResponse struct {
	NewUsersToday  int64             `json:"newUsersToday"`  // 今日新增用户
	TodaySales     float64           `json:"todaySales"`     // 今日销售额
	MonthSales     float64           `json:"monthSales"`     // 本月销售额
	ProductRanking []ProductRankItem `json:"productRanking"` // 商品销售排行
}

type ProductRankItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	TotalSales  int     `json:"totalSales"`  // 销售数量
	TotalAmount float64 `json:"totalAmount"` // 销售金额
}

// TrendRequest 趋势请求参数
type TrendRequest struct {
	Type string `form:"type"` // 维度
}

// TrendResponse 趋势数据
type TrendResponse struct {
	Dates  []string  `json:"dates"`  // 日期标签
	Orders []int64   `json:"orders"` // 订单量
	Sales  []float64 `json:"sales"`  // 销售额
}
