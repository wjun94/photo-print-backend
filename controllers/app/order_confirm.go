package app

import (
	"fmt"
	"log"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/services"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PreviewOrderItem struct {
	ImageURL string `json:"imageUrl" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type PreviewOrderReq struct {
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
}

// PreviewOrder 确认订单页面预览（获取商品信息、默认地址）
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
		var spec models.ProductSpec
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
			utils.Fail(c, "规格["+spec.Name+"]库存不足")
			return
		}

		subtotal := float64(it.Quantity) * spec.Price
		totalAmount += subtotal

		// 添加到单个图片项列表
		previewItems = append(previewItems, PreviewItemResponse{
			ProductID:   product.ID.String(),
			ProductName: product.Name,
			SpecID:      spec.ID.String(),
			SpecName:    spec.Name,
			Price:       spec.Price,
			Quantity:    it.Quantity,
			Subtotal:    subtotal,
			ImageURL:    it.ImageURL, // 注意：这里修正为使用item的图片URL，而不是商品封面
		})

		// 更新规格汇总
		specIDStr := spec.ID.String()
		if summary, exists := specSummaryMap[specIDStr]; exists {
			// 规格已存在，累加数量和小计
			summary.TotalQuantity += it.Quantity
			summary.TotalSubtotal += subtotal
		} else {
			// 规格不存在，新建汇总条目
			specSummaryMap[specIDStr] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        specIDStr,
				SpecName:      spec.Name,
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
		// 填充名称（如需）
		addressResp = AddressListResponse{
			Address:      defaultAddress,
			ProvinceName: utils.GetRegionName(defaultAddress.ProvinceID),
			CityName:     utils.GetRegionName(defaultAddress.CityID),
			DistrictName: utils.GetRegionName(defaultAddress.DistrictID),
		}
	} else {
		addressResp = nil
	}

	// 在计算 totalAmount 之后
	freight := utils.CalculateFreight(totalAmount)
	actualAmount := totalAmount + freight

	utils.Success(c, gin.H{
		"items":          previewItems,
		"specs":          specSummaries, // 新增：不重复的规格汇总数组
		"totalAmount":    totalAmount,
		"freight":        freight,
		"actualAmount":   actualAmount,
		"defaultAddress": addressResp,
	})
}

// 提交订单请求
type SubmitOrderReq struct {
	AddressId string             `json:"addressId" binding:"required"`
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
	Remark    string             `json:"remark"` // 可选备注
}

// SubmitOrder 立即购买提交订单
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
		spec     models.ProductSpec
		qty      int
		amount   float64
		imageURL string // 用户上传的图片

	}
	checks := make([]itemCheck, 0, len(req.Items))
	var totalAmount float64

	for _, it := range req.Items {
		productID, _ := strconv.ParseInt(req.ProductID, 10, 64)
		specID, _ := strconv.ParseInt(req.SpecID, 10, 64)

		var spec models.ProductSpec
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
			utils.Fail(c, "规格["+spec.Name+"]库存不足，当前库存: "+strconv.Itoa(spec.Stock))
			return
		}

		subtotal := float64(it.Quantity) * spec.Price
		totalAmount += subtotal
		checks = append(checks, itemCheck{spec: spec, imageURL: it.ImageURL, qty: it.Quantity, amount: subtotal})
	}

	// 生成订单号
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())

	// 组装地址字符串（可根据需要拼接省市区）
	fullAddress := address.Detail
	if address.Doorplate != "" {
		fullAddress += " " + address.Doorplate
	}

	// 计算运费和实付
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

	// 创建订单地址快照
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
	// ====================== 事务创建订单（原子性保证） ======================
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 创建订单
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		orderAddress.OrderID = order.ID
		// 2. 创建订单地址快照
		if err := tx.Create(&orderAddress).Error; err != nil {
			utils.Fail(c, fmt.Sprintf("创建订单地址失败: %s", err.Error()))
		}
		// 创建订单项并扣减库存
		for _, ch := range checks {
			// 扣减库存（原子操作）
			newStock := ch.spec.Stock - ch.qty
			if err := tx.Model(&ch.spec).Update("stock", newStock).Error; err != nil {
				return err
			}
			item := models.OrderItem{
				OrderID:  order.ID,
				ImageURL: ch.imageURL, // 使用商品主图
				Spec:     ch.spec.Name,
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

	// 只有已发货的订单才能确认收货
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

	// 在 order.Status 变为 models.OrderStatusCompleted 后
	if err := services.CreateCommissionForOrder(order); err != nil {
		// 记录错误日志，但不要影响主流程
		log.Printf("生成佣金失败, orderId=%s, err=%v", order.ID.String(), err)
	}

	utils.Success(c, nil)
}
