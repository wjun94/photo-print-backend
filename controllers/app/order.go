package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"gorm.io/gorm"

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
				ImageURL:      it.ImageURL,
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

// CreateOrder 创建订单
// @Summary 创建订单
// @Description 用户下单打印照片
// @Tags 订单
// @Accept json
// @Produce json
// @Param order body CreateOrderReq true "订单信息"
// @Success 200 {object} utils.Response{data=models.Order}
// @Router /api/v1/orders [post]
func CreateOrder(c *gin.Context) {
	// 使用辅助函数获取用户ID（int64）
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录或用户ID无效")
		return
	}

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 计算总金额
	var total float64
	for _, it := range req.Items {
		total += float64(it.Quantity) * it.Price
	}
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())
	order := models.Order{
		OrderNo: orderNo,
		UserID:  utils.Int64Str(userID),
		Address: req.Address,
		Status:  "pending",
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for _, it := range req.Items {
			item := models.OrderItem{
				OrderID:  order.ID,
				ImageURL: it.ImageURL,
				Spec:     it.Spec,
				Quantity: it.Quantity,
				Price:    it.Price,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "创建订单失败")
		return
	}
	database.DB.Preload("Items").First(&order, order.ID)
	utils.Success(c, order)
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
	if err := database.DB.Preload("Items").First(&order, id).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}
	summaries, _ := buildOrderSpecSummaries(order.ID.Int64())
	order.Specs = summaries
	utils.Success(c, order)
}

// 微信订单列表返回数据
type OrderListResult struct {
	ID           utils.Int64Str               `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderNo      string                       `gorm:"uniqueIndex;size:32;not null" json:"orderNo"`
	Status       string                       `gorm:"default:'pending';size:20" json:"status"`
	ActualAmount float64                      `gorm:"type:decimal(10,2);not null" json:"actualAmount"` // 实付 = amount + freight
	CreatedAt    utils.LocalTime              `json:"createdAt"`
	Specs        []models.SpecSummaryResponse `gorm:"-" json:"specs"`
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
	query := database.DB.Model(&models.Order{}).Where("user_id = ?", userID).Preload("Items")
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
