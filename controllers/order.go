package controllers

import (
	"fmt"
	"gorm.io/gorm"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateOrderItem struct {
	PhotoID  uint    `json:"photo_id" binding:"required"`
	Spec     string  `json:"spec" binding:"required"`
	Quantity int     `json:"quantity" binding:"required,min=1"`
	Price    float64 `json:"price" binding:"required,gt=0"`
}

type CreateOrderReq struct {
	UserID  string            `json:"user_id" binding:"required"`
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
	// 获取用户ID
	userIDVal, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, "未登录")
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		utils.Fail(c, "用户身份无效")
		return
	}

	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 校验照片是否存在
	for _, it := range req.Items {
		var photo models.Photo
		if err := database.DB.First(&photo, it.PhotoID).Error; err != nil {
			utils.Fail(c, fmt.Sprintf("照片ID %d 不存在", it.PhotoID))
			return
		}
		// 可选：检查照片是否属于当前用户（权限校验）
		if photo.UserID != userID {
			utils.Fail(c, fmt.Sprintf("照片ID %d 不属于当前用户", it.PhotoID))
			return
		}
	}

	// 计算总金额
	var total float64
	for _, it := range req.Items {
		total += float64(it.Quantity) * it.Price
	}
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())
	order := models.Order{
		OrderNo:     orderNo,
		UserID:      userID,
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
				PhotoID:  it.PhotoID,
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
