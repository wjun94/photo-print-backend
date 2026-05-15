package controllers

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
	ImageURL string  `json:"image_url" binding:"required"`
	Spec     string  `json:"spec" binding:"required"`
	Quantity int     `json:"quantity" binding:"required,min=1"`
	Price    float64 `json:"price" binding:"required,gt=0"`
}

type CreateOrderReq struct {
	Address string            `json:"address" binding:"required"`
	Items   []CreateOrderItem `json:"items" binding:"required,min=1"`
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
		OrderNo:     orderNo,
		UserID:      utils.Int64Str(userID),
		Address:     req.Address,
		TotalAmount: total,
		Status:      "pending",
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
	database.DB.Preload("Items").Preload("Items.Photo").First(&order, order.ID)
	utils.Success(c, order)
}

// GetOrderList 订单列表 (后台)
// @Summary 订单列表
// @Description 后台查看订单，支持分页和状态筛选
// @Tags 订单
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param status query string false "状态"
// @Success 200 {object} utils.Response{data=[]models.Order}
// @Router /api/v1/orders [get]
func GetOrderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	status := c.Query("status")
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size
	var orders []models.Order
	query := database.DB.Model(&models.Order{}).Preload("Items").Preload("Items.Photo")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&orders)
	utils.Success(c, gin.H{
		"list":  orders,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetOrderDetail 订单详情
// @Summary 订单详情
// @Tags 订单
// @Param id path int true "订单ID"
// @Success 200 {object} utils.Response{data=models.Order}
// @Router /api/v1/orders/{id} [get]
func GetOrderDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var order models.Order
	if err := database.DB.Preload("Items").Preload("Items.Photo").First(&order, id).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}
	utils.Success(c, order)
}

// UpdateOrderStatus 更新订单状态 (后台)
// @Summary 更新订单状态
// @Tags 订单
// @Param id path int true "订单ID"
// @Param status body object true "状态字段"
// @Success 200 {object} utils.Response
// @Router /api/v1/orders/{id}/status [put]
func UpdateOrderStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=pending paid processing completed cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "状态值无效: "+err.Error())
		return
	}
	result := database.DB.Model(&models.Order{}).Where("id = ?", id).Update("status", req.Status)
	if result.RowsAffected == 0 {
		utils.Fail(c, "订单不存在")
		return
	}
	utils.Success(c, nil)
}

// GetMyOrders 获取当前小程序用户的订单列表
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
	var total int64
	query := database.DB.Model(&models.Order{}).Where("user_id = ?", userID).Preload("Items").Preload("Items.Photo")
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&orders)

	utils.Success(c, gin.H{
		"list":  orders,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
