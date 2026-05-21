package admin

import (
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
	query := database.DB.Model(&models.Order{}).Preload("Items")
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
	if err := database.DB.Preload("Items").First(&order, id).Error; err != nil {
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
