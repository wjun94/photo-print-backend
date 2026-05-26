package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PreviewOrderReq struct {
	ProductID string `json:"productId" binding:"required"`
	SpecID    string `json:"specId" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

// PreviewOrder 确认订单页面预览（获取商品信息、默认地址）
func PreviewOrder(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req PreviewOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	// 查询商品规格
	var spec models.ProductSpec
	specID, _ := strconv.ParseInt(req.SpecID, 10, 64)
	if err := database.DB.Preload("Product").First(&spec, specID).Error; err != nil {
		utils.Fail(c, "规格不存在")
		return
	}

	// 商品信息
	product := spec.Product
	if product.Status != models.ProductStatusOnSale {
		utils.Fail(c, "商品已下架")
		return
	}

	// 库存检查
	if spec.Stock < req.Quantity {
		utils.Fail(c, "库存不足")
		return
	}

	// 查询用户默认地址
	var defaultAddress models.Address
	err := database.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&defaultAddress).Error
	// 如果没有默认地址，允许为空，前端可提示新增地址
	address := &defaultAddress
	if err != nil {
		address = nil
	}

	// 构建预览数据
	preview := gin.H{
		"product": gin.H{
			"id":          product.ID.String(),
			"name":        product.Name,
			"coverImage":  product.CoverImage,
			"description": product.Description,
		},
		"spec": gin.H{
			"id":    spec.ID.String(),
			"name":  spec.Name,
			"price": spec.Price,
			"stock": spec.Stock,
		},
		"quantity":    req.Quantity,
		"totalAmount": float64(req.Quantity) * spec.Price,
		"address":     address, // 可能为 null
	}

	utils.Success(c, preview)
}

// SubmitOrderDirect 立即购买提交订单
func SubmitOrderDirect(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req struct {
		AddressID string `json:"addressId" binding:"required"`
		ProductID string `json:"productId" binding:"required"`
		SpecID    string `json:"specId" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required,min=1"`
		Remark    string `json:"remark"` // 可选备注
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	// 查询地址
	addressID, _ := strconv.ParseInt(req.AddressID, 10, 64)
	var address models.Address
	if err := database.DB.First(&address, addressID).Error; err != nil {
		utils.Fail(c, "地址不存在")
		return
	}
	if address.UserID.Int64() != userID {
		utils.Fail(c, "地址不属于当前用户")
		return
	}

	// 查询商品规格
	specID, _ := strconv.ParseInt(req.SpecID, 10, 64)
	var spec models.ProductSpec
	if err := database.DB.Preload("Product").First(&spec, specID).Error; err != nil {
		utils.Fail(c, "规格不存在")
		return
	}
	product := spec.Product
	if product.Status != models.ProductStatusOnSale {
		utils.Fail(c, "商品已下架")
		return
	}
	if spec.Stock < req.Quantity {
		utils.Fail(c, "库存不足")
		return
	}

	// 构建订单数据
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())
	totalAmount := float64(req.Quantity) * spec.Price

	order := models.Order{
		OrderNo: orderNo,
		UserID:  utils.Int64Str(userID),
		Address: address.Detail, // 可根据需要组装完整地址
		Amount:  totalAmount,
		Status:  "pending",
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 创建订单
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		// 扣减库存
		newStock := spec.Stock - req.Quantity
		if err := tx.Model(&spec).Update("stock", newStock).Error; err != nil {
			return err
		}
		// 创建订单项
		item := models.OrderItem{
			OrderID:  order.ID,
			ImageURL: product.CoverImage,
			Spec:     spec.Name,
			Quantity: req.Quantity,
			Price:    spec.Price,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		utils.Fail(c, "下单失败，请重试")
		return
	}

	// 返回订单ID
	utils.Success(c, gin.H{
		"orderId": order.ID.String(),
	})
}
