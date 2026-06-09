package app

import (
	"fmt"
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/services"
	"photo-print-backend/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 辅助函数：获取规格显示名称（优先从 Attributes 拼接，否则使用 SkuKey）
func getSpecDisplayName(spec models.Spec) string {
	if len(spec.Attributes) > 0 {
		var values []string
		// 按照属性顺序拼接，通常前端需要顺序一致，这里简单遍历 map 顺序不定
		// 可按需根据商品属性模板排序，此处仅演示
		for _, v := range spec.Attributes {
			values = append(values, v)
		}
		return strings.Join(values, " ")
	}
	return spec.SkuKey
}

// PreviewOrderItem 预览订单项
type PreviewOrderItem struct {
	ImageURL string `json:"imageUrl" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// PreviewOrderReq 预览订单请求
type PreviewOrderReq struct {
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
}

// PreviewItemResponse 预览项响应
type PreviewItemResponse struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	SpecID      string  `json:"specId"`
	SpecName    string  `json:"specName"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	Subtotal    float64 `json:"subtotal"`
	ImageURL    string  `json:"imageUrl"`
}

// PreviewOrder 确认订单页面预览
// @Summary 订单预览
// @Description 根据商品、规格、数量计算总金额，并返回默认地址
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body PreviewOrderReq true "预览请求"
// @Success 200 {object} utils.Response{data=object{items=[]PreviewItemResponse,specs=[]models.SpecSummaryResponse,totalAmount=float64,freight=float64,actualAmount=float64,defaultAddress=object}}
// @Router /api/v1/wx/order/preview [post]
func PreviewOrder(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req PreviewOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	var previewItems []PreviewItemResponse
	var totalAmount float64

	// 用于统计规格汇总的map，key为specID字符串
	specSummaryMap := make(map[string]*models.SpecSummaryResponse)

	// 逐个验证商品规格
	for _, it := range req.Items {
		productID, err := strconv.ParseInt(req.ProductID, 10, 64)
		if err != nil {
			utils.Fail(c, "无效的商品ID: "+req.ProductID)
			return
		}
		specID, err := strconv.ParseInt(req.SpecID, 10, 64)
		if err != nil {
			utils.Fail(c, "无效的规格ID: "+req.SpecID)
			return
		}

		// 查询规格（预加载商品）
		var spec models.Spec
		if err := database.DB.Preload("Product").First(&spec, specID).Error; err != nil {
			utils.Fail(c, "规格不存在: "+req.SpecID)
			return
		}
		product := spec.Product
		if product.ID.Int64() != productID {
			utils.Fail(c, "商品与规格不匹配")
			return
		}
		if product.Status != models.ProductStatusOnSale {
			utils.Fail(c, "商品["+product.Name+"]已下架")
			return
		}
		if spec.Stock < it.Quantity {
			utils.Fail(c, "规格库存不足")
			return
		}

		subtotal := float64(it.Quantity) * spec.Price
		totalAmount += subtotal

		// 添加到单个图片项列表
		previewItems = append(previewItems, PreviewItemResponse{
			ProductID:   product.ID.String(),
			ProductName: product.Name,
			SpecID:      spec.ID.String(),
			SpecName:    getSpecDisplayName(spec),
			Price:       spec.Price,
			Quantity:    it.Quantity,
			Subtotal:    subtotal,
			ImageURL:    it.ImageURL,
		})

		// 更新规格汇总
		specIDStr := spec.ID.String()
		if summary, exists := specSummaryMap[specIDStr]; exists {
			summary.TotalQuantity += it.Quantity
			summary.TotalSubtotal += subtotal
		} else {
			specSummaryMap[specIDStr] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        specIDStr,
				SpecName:      getSpecDisplayName(spec),
				Price:         spec.Price,
				TotalQuantity: it.Quantity,
				TotalSubtotal: subtotal,
				ImageURL:      it.ImageURL,
			}
		}
	}

	// 将map转换为数组
	var specSummaries []models.SpecSummaryResponse
	for _, summary := range specSummaryMap {
		specSummaries = append(specSummaries, *summary)
	}

	// 获取用户默认地址
	var defaultAddress models.Address
	addrErr := database.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&defaultAddress).Error
	var addressResp interface{}
	if addrErr == nil {
		addressResp = struct {
			ID           string `json:"id"`
			ReceiverName string `json:"receiverName"`
			Mobile       string `json:"mobile"`
			ProvinceName string `json:"provinceName"`
			CityName     string `json:"cityName"`
			DistrictName string `json:"districtName"`
			Detail       string `json:"detail"`
			Doorplate    string `json:"doorplate"`
		}{
			ID:           defaultAddress.ID.String(),
			ReceiverName: defaultAddress.ReceiverName,
			Mobile:       defaultAddress.Mobile,
			ProvinceName: utils.GetRegionName(defaultAddress.ProvinceID),
			CityName:     utils.GetRegionName(defaultAddress.CityID),
			DistrictName: utils.GetRegionName(defaultAddress.DistrictID),
			Detail:       defaultAddress.Detail,
			Doorplate:    defaultAddress.Doorplate,
		}
	} else {
		addressResp = nil
	}

	freight := utils.CalculateFreight(totalAmount)
	actualAmount := totalAmount + freight

	utils.Success(c, gin.H{
		"items":          previewItems,
		"specs":          specSummaries,
		"totalAmount":    totalAmount,
		"freight":        freight,
		"actualAmount":   actualAmount,
		"defaultAddress": addressResp,
	})
}

// SubmitOrderReq 提交订单请求
type SubmitOrderReq struct {
	AddressId string             `json:"addressId" binding:"required"`
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
	Remark    string             `json:"remark"`
}

// SubmitOrder 立即购买提交订单
// @Summary 提交订单
// @Description 立即购买，创建订单并扣减库存
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body SubmitOrderReq true "订单信息"
// @Success 200 {object} utils.Response{data=object{orderId=string}}
// @Router /api/v1/wx/order/submit [post]
func SubmitOrder(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req SubmitOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 验证地址
	addressID, _ := strconv.ParseInt(req.AddressId, 10, 64)
	var address models.Address
	if err := database.DB.First(&address, addressID).Error; err != nil {
		utils.Fail(c, "地址不存在")
		return
	}
	if address.UserID.Int64() != userID {
		utils.Fail(c, "地址不属于当前用户")
		return
	}

	// 预检：收集所有商品规格信息，验证库存和价格
	type itemCheck struct {
		spec     models.Spec
		qty      int
		amount   float64
		imageURL string
	}
	checks := make([]itemCheck, 0, len(req.Items))
	var totalAmount float64

	for _, it := range req.Items {
		productID, _ := strconv.ParseInt(req.ProductID, 10, 64)
		specID, _ := strconv.ParseInt(req.SpecID, 10, 64)

		var spec models.Spec
		if err := database.DB.Preload("Product").First(&spec, specID).Error; err != nil {
			utils.Fail(c, "规格不存在: "+req.SpecID)
			return
		}
		if spec.Product.ID.Int64() != productID {
			utils.Fail(c, "商品与规格不匹配")
			return
		}
		if spec.Product.Status != models.ProductStatusOnSale {
			utils.Fail(c, "商品["+spec.Product.Name+"]已下架")
			return
		}
		if spec.Stock < it.Quantity {
			utils.Fail(c, "规格库存不足，当前库存: "+strconv.Itoa(spec.Stock))
			return
		}

		subtotal := float64(it.Quantity) * spec.Price
		totalAmount += subtotal
		checks = append(checks, itemCheck{
			spec:     spec,
			qty:      it.Quantity,
			amount:   subtotal,
			imageURL: it.ImageURL,
		})
	}

	// 生成订单号
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())

	// 组装地址字符串
	fullAddress := address.Detail
	if address.Doorplate != "" {
		fullAddress += " " + address.Doorplate
	}

	freight := utils.CalculateFreight(totalAmount)
	actualAmount := totalAmount + freight

	order := models.Order{
		OrderNo:      orderNo,
		UserID:       utils.Int64Str(userID),
		Amount:       totalAmount,
		Freight:      freight,
		ActualAmount: actualAmount,
		Remark:       req.Remark,
		Status:       models.OrderStatusPending,
	}

	// 订单地址快照
	orderAddress := models.OrderAddress{
		ReceiverName: address.ReceiverName,
		Mobile:       address.Mobile,
		ProvinceID:   address.ProvinceID,
		ProvinceName: utils.GetRegionName(address.ProvinceID),
		CityID:       address.CityID,
		CityName:     utils.GetRegionName(address.CityID),
		DistrictID:   address.DistrictID,
		DistrictName: utils.GetRegionName(address.DistrictID),
		Detail:       address.Detail,
		Doorplate:    address.Doorplate,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		orderAddress.OrderID = order.ID
		if err := tx.Create(&orderAddress).Error; err != nil {
			return err
		}
		for _, ch := range checks {
			// 扣减库存
			newStock := ch.spec.Stock - ch.qty
			if err := tx.Model(&ch.spec).Update("stock", newStock).Error; err != nil {
				return err
			}
			item := models.OrderItem{
				OrderID:  order.ID,
				ImageURL: ch.imageURL,
				Spec:     getSpecDisplayName(ch.spec),
				SpecID:   ch.spec.ID,
				Quantity: ch.qty,
				Price:    ch.spec.Price,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		utils.Fail(c, "创建订单失败，请重试")
		return
	}

	utils.Success(c, gin.H{
		"orderId": order.ID.String(),
	})
}

// ConfirmReceipt 用户确认收货
// @Summary 确认收货
// @Description 用户确认收货后订单状态变为已完成，并生成佣金
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body object{orderId=string} true "订单ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/wx/order/confirm [post]
func ConfirmReceipt(c *gin.Context) {
	var req struct {
		OrderID string `json:"orderId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	orderID, err := strconv.ParseInt(req.OrderID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效订单ID")
		return
	}

	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}

	if order.Status != models.OrderStatusShipped {
		utils.Fail(c, "订单未发货，无法确认收货")
		return
	}

	now := utils.LocalTime(time.Now())
	order.Status = models.OrderStatusCompleted
	order.FinishAt = &now

	if err := database.DB.Save(&order).Error; err != nil {
		utils.Fail(c, "确认收货失败")
		return
	}

	// 生成佣金
	if err := services.CreateCommissionForOrder(order); err != nil {
		log.Printf("生成佣金失败, orderId=%s, err=%v", order.ID.String(), err)
	}

	utils.Success(c, nil)
}
