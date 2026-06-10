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
	ImageURL string `json:"imageUrl"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type PreviewOrderReq struct {
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
	CouponID  string             `json:"couponId"` // 新增：优惠券ID（可选）
}

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

// PreviewOrder 确认订单页面预览（支持优惠券）
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
	specSummaryMap := make(map[string]*models.SpecSummaryResponse)

	// 验证商品规格（与原逻辑相同）
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

		imageURL := it.ImageURL
		if product.Action == models.ProductActionUpload {
			if imageURL == "" {
				utils.Fail(c, "该商品需要上传图片，请提供图片URL")
				return
			}
		} else {
			if imageURL == "" {
				imageURL = product.CoverImage
			}
		}

		subtotal := float64(it.Quantity) * spec.Price
		totalAmount += subtotal

		previewItems = append(previewItems, PreviewItemResponse{
			ProductID:   product.ID.String(),
			ProductName: product.Name,
			SpecID:      spec.ID.String(),
			SpecName:    services.GetSpecDisplayName(spec),
			Price:       spec.Price,
			Quantity:    it.Quantity,
			Subtotal:    subtotal,
			ImageURL:    imageURL,
		})

		specIDStr := spec.ID.String()
		if summary, exists := specSummaryMap[specIDStr]; exists {
			summary.TotalQuantity += it.Quantity
			summary.TotalSubtotal += subtotal
		} else {
			specSummaryMap[specIDStr] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        specIDStr,
				SpecName:      services.GetSpecDisplayName(spec),
				Price:         spec.Price,
				TotalQuantity: it.Quantity,
				TotalSubtotal: subtotal,
				ImageURL:      imageURL,
			}
		}
	}

	var specSummaries []models.SpecSummaryResponse
	for _, summary := range specSummaryMap {
		specSummaries = append(specSummaries, *summary)
	}

	// 获取默认地址
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
	discountAmount := 0.0

	// 优惠券处理（预览）
	if req.CouponID != "" {
		couponID, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err == nil {
			discount, _, err := services.ValidateCoupon(couponID, userID, totalAmount+freight)
			if err == nil {
				discountAmount = discount
				actualAmount = totalAmount + freight - discount
				if actualAmount < 0 {
					actualAmount = 0
				}
			} else {
				// 优惠券无效，返回错误信息（可根据需求决定是否阻断）
				utils.Fail(c, err.Error())
				return
			}
		}
	}

	utils.Success(c, gin.H{
		"items":          previewItems,
		"specs":          specSummaries,
		"totalAmount":    totalAmount,
		"freight":        freight,
		"discountAmount": discountAmount,
		"actualAmount":   actualAmount,
		"defaultAddress": addressResp,
	})
}

// SubmitOrderReq 提交订单请求（增加优惠券ID）
type SubmitOrderReq struct {
	AddressId string             `json:"addressId" binding:"required"`
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
	Remark    string             `json:"remark"`
	CouponID  string             `json:"couponId"`
}

// SubmitOrder 提交订单（支持优惠券）
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

	// 预检商品规格
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

	// 计算运费
	freight := utils.CalculateFreight(totalAmount)
	actualAmount := totalAmount + freight
	discountAmount := 0.0
	var usedCouponID int64

	// 优惠券校验（如果提供）
	if req.CouponID != "" {
		cid, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err == nil {
			discount, _, err := services.ValidateCoupon(cid, userID, totalAmount+freight)
			if err != nil {
				utils.Fail(c, err.Error())
				return
			}
			discountAmount = discount
			actualAmount = totalAmount + freight - discount
			if actualAmount < 0 {
				actualAmount = 0
			}
			usedCouponID = cid
		}
	}

	// 生成订单号
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())
	fullAddress := address.Detail
	if address.Doorplate != "" {
		fullAddress += " " + address.Doorplate
	}

	order := models.Order{
		OrderNo:        orderNo,
		UserID:         utils.Int64Str(userID),
		Amount:         totalAmount,
		Freight:        freight,
		DiscountAmount: discountAmount,
		ActualAmount:   actualAmount,
		Remark:         req.Remark,
		Status:         models.OrderStatusPending,
	}
	if usedCouponID != 0 {
		order.CouponID = utils.Int64Str(usedCouponID)
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
		// 创建订单
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		orderAddress.OrderID = order.ID
		if err := tx.Create(&orderAddress).Error; err != nil {
			return err
		}
		// 扣减库存，保存订单项
		for _, ch := range checks {
			newStock := ch.spec.Stock - ch.qty
			if err := tx.Model(&ch.spec).Update("stock", newStock).Error; err != nil {
				return err
			}
			item := models.OrderItem{
				OrderID:  order.ID,
				ImageURL: ch.imageURL,
				Spec:     services.GetSpecDisplayName(ch.spec),
				SpecID:   ch.spec.ID,
				Quantity: ch.qty,
				Price:    ch.spec.Price,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		// 如果使用了优惠券，更新用户优惠券状态
		if usedCouponID != 0 {
			if err := tx.Model(&models.UserCoupon{}).
				Where("user_id = ? AND coupon_id = ? AND status = ?", userID, usedCouponID, models.UserCouponUnused).
				Updates(map[string]interface{}{
					"status":   models.UserCouponUsed,
					"order_id": order.ID,
					"used_at":  time.Now(),
				}).Error; err != nil {
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
