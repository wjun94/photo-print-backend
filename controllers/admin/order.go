package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/services"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetOrderList 订单列表 (后台)
// @Summary 订单列表
// @Description 后台查看订单，支持分页、状态筛选、订单号模糊查询、创建时间区间查询
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param status query string false "状态"
// @Param order_no query string false "订单号（模糊匹配）"
// @Param createdAtStart query string false "创建时间起始，格式 2006-01-02 15:04:05"
// @Param createdAtEnd query string false "创建时间结束，格式 2006-01-02 15:04:05"
// @Success 200 {object} utils.Response{data=object{list=[]models.Order,total=int64,page=int,size=int}}
// @Router /api/v1/admin/orders [get]
func GetOrderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	status := c.Query("status")
	orderNo := c.Query("orderNo")
	createdAtStart := c.Query("createdAtStart")
	createdAtEnd := c.Query("createdAtEnd")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size

	var orders []models.Order
	var total int64
	query := database.DB.Model(&models.Order{}).Preload("Address").Preload("Logistics")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if orderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+orderNo+"%")
	}
	if createdAtStart != "" {
		query = query.Where("created_at >= ?", createdAtStart)
	}
	if createdAtEnd != "" {
		query = query.Where("created_at <= ?", createdAtEnd)
	}

	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&orders)

	for i := range orders {
		summaries, err := services.BuildOrderSpecSummaries(orders[i].ID.Int64())
		if err == nil {
			orders[i].Specs = summaries
		}
	}

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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var order models.Order
	if err := database.DB.Preload("Items.SpecInfo.Product").Preload("Address").Preload("Logistics").First(&order, id).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}

	// 过滤订单项：只保留商品 action 为 upload 的项
	filteredItems := make([]models.OrderItem, 0)
	for _, item := range order.Items {
		if item.SpecInfo.Product.Action == models.ProductActionUpload {
			filteredItems = append(filteredItems, item)
		}
	}
	order.Items = filteredItems

	// 生成规格汇总（该函数内部已只汇总 upload 商品，无需改动）
	summaries, err := services.BuildOrderSpecSummaries(order.ID.Int64())
	if err == nil {
		order.Specs = summaries
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
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
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
