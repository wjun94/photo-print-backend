package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateOrderItem struct {
	ImageURL string  `json:"imageUrl" binding:"required"`
	Spec     string  `json:"spec" binding:"required"`
	Quantity int     `json:"quantity" binding:"required,min=1"`
	Price    float64 `json:"price" binding:"required,gt=0"`
}

type CreateOrderReq struct {
	Address string            `json:"address" binding:"required"`
	Items   []CreateOrderItem `json:"items" binding:"required,min=1"`
}

// buildOrderSpecSummaries 基于 spec 字段分组汇总
func buildOrderSpecSummaries(orderID int64) ([]models.SpecSummaryResponse, error) {
	var items []models.OrderItem
	// 预加载规格和商品
	err := database.DB.
		Where("order_id = ?", orderID).
		Preload("SpecInfo.Product").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	// 按 SpecID 分组汇总
	group := make(map[int64]*models.SpecSummaryResponse)
	for _, it := range items {
		spec := it.SpecInfo
		product := spec.Product
		specID := spec.ID.Int64()
		if _, ok := group[specID]; !ok {
			group[specID] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        spec.ID.String(),
				SpecName:      spec.Name,
				Price:         it.Price, // 订单项中的价格（可能与规格当前价格不同，但以订单为准）
				TotalQuantity: 0,
				TotalSubtotal: 0,
				ImageURL:      product.CoverImage,
			}
		}
		group[specID].TotalQuantity += it.Quantity
		group[specID].TotalSubtotal += float64(it.Quantity) * it.Price
	}

	summaries := make([]models.SpecSummaryResponse, 0, len(group))
	for _, v := range group {
		summaries = append(summaries, *v)
	}
	return summaries, nil
}

// GetOrderDetail 订单详情
// @Summary 订单详情
// @Tags 订单
// @Param id path int true "订单ID"
// @Success 200 {object} utils.Response{data=models.Order}
// @Router /api/v1/orders/{id} [get]
func GetOrderDetail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var order models.Order
	if err := database.DB.Preload("Address").First(&order, id).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}
	summaries, _ := buildOrderSpecSummaries(order.ID.Int64())
	order.Specs = summaries
	utils.Success(c, order)
}

// 微信订单列表返回数据
type OrderListResult struct {
	ID           utils.Int64Str               `json:"id"`
	OrderNo      string                       `json:"orderNo"`
	Status       string                       `json:"status"`
	ActualAmount float64                      `json:"actualAmount"` // 实付 = amount + freight
	CreatedAt    utils.LocalTime              `json:"createdAt"`
	Specs        []models.SpecSummaryResponse `json:"specs"`
}

// GetWxOrders 获取当前小程序用户的订单列表
// @Summary 获取我的订单
// @Tags 订单
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} utils.Response{data=object{list=[]models.Order,total=int64,page=int,size=int}}
// @Router /api/v1/orders/my [get]
func GetWxOrders(c *gin.Context) {
	// 从上下文中获取用户ID（由 AuthMiddleware 设置）
	userIDVal, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未登录")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		utils.Unauthorized(c, "用户ID无效")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size
	var orders []models.Order

	var result []OrderListResult
	var total int64
	query := database.DB.Model(&models.Order{}).Where("user_id = ?", userID)
	// 增加 status 查询条件
	statusStr := c.Query("status")
	fmt.Println("----11111111144444")
	fmt.Println(statusStr)
	if statusStr != "" {
		query = query.Where("status = ?", statusStr)
	} else {
		// 未传 status 时，默认排除已取消的订单
		query = query.Where("status != ?", models.OrderStatusCancelled)
	}
	query.Preload("Items")
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&orders)

	// 为每个订单构建 SpecSummaries
	for i := range orders {
		summaries, err := buildOrderSpecSummaries(orders[i].ID.Int64())
		item := OrderListResult{
			ID:        orders[i].ID,
			OrderNo:   orders[i].OrderNo,
			Status:    orders[i].Status,
			CreatedAt: orders[i].CreatedAt,
		}
		result = append(result, item)
		if err == nil {
			result[i].Specs = summaries
		}
	}

	utils.Success(c, gin.H{
		"list":  result,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
