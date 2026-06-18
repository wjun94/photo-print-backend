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

// SubmitOrderItem 提交订单商品项
type SubmitOrderItem struct {
	Quantity int    `json:"quantity" binding:"required,min=1"`
	ImageURL string `json:"imageUrl"`
}

// SubmitOrderReq 提交订单请求（增加优惠券ID）
type SubmitOrderReq struct {
	AddressId string            `json:"addressId" binding:"required"`
	ProductID string            `json:"productId" binding:"required"`
	SpecID    string            `json:"specId" binding:"required"`
	Items     []SubmitOrderItem `json:"items" binding:"required,min=1"`
	Remark    string            `json:"remark"`
	CouponID  string            `json:"couponId"`
}

// SubmitOrder 提交订单（支持优惠券）
// @Summary 提交订单
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body SubmitOrderReq true "订单信息"
// @Success 200 {object} utils.Response{data=object{orderId string}}
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

	// ---------- 1. 验证地址 ----------
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

	// ---------- 2. 预检商品规格、计算总金额 ----------
	type itemCheck struct {
		spec     models.Spec
		qty      int
		amount   float64
		imageURL string
	}
	checks := make([]itemCheck, 0, len(req.Items))
	var totalAmount float64

	productID, _ := strconv.ParseInt(req.ProductID, 10, 64)
	specID, _ := strconv.ParseInt(req.SpecID, 10, 64)

	for _, it := range req.Items {
		var spec models.Spec
		if err := database.DB.Preload("Product").First(&spec, specID).Error; err != nil {
			utils.Fail(c, "规格不存在: "+req.SpecID)
			return
		}
		var product models.Product
		if err := database.DB.First(&product, spec.ProductID).Error; err != nil {
			utils.Fail(c, "商品不存在")
			return
		}
		if product.ID.Int64() != productID {
			utils.Fail(c, "商品与规格不匹配")
			return
		}
		if product.Status != models.ProductStatusOnSale {
			utils.Fail(c, "商品["+product.Name+"]已下架")
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

	// ---------- 3. 计算运费 ----------
	freight := utils.CalculateFreight(totalAmount)
	orderTotal := totalAmount + freight

	// ---------- 4. 优惠券处理 ----------
	discountAmount := 0.0
	var usedUserCouponID int64 // 用户券实例ID

	if req.CouponID != "" {
		// 解析传入的ID
		inputID, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err != nil {
			utils.Fail(c, "优惠券ID格式错误")
			return
		}

		var userCoupon models.UserCoupon
		var errQuery error

		// 优先尝试作为用户券实例ID查询
		errQuery = database.DB.Preload("Coupon").
			Where("id = ? AND user_id = ? AND status = ?", inputID, userID, models.UserCouponUnused).
			First(&userCoupon).Error

		// 若未找到，尝试作为模板ID查询（从当前用户持有的该模板券中选择最优的一张）
		if errQuery != nil {
			// 查询该模板下用户持有的所有未使用且未过期的券，按优惠金额最大或有效期最近排列
			now := time.Now()
			var candidates []models.UserCoupon
			if err := database.DB.Preload("Coupon").
				Where("coupon_id = ? AND user_id = ? AND status = ? AND valid_start <= ? AND valid_end >= ?",
							inputID, userID, models.UserCouponUnused, now, now).
				Order("valid_end ASC"). // 优先使用即将过期的券
				First(&candidates).Error; err == nil && len(candidates) > 0 {
				userCoupon = candidates[0]
				errQuery = nil
			}
		}

		if errQuery != nil {
			utils.Fail(c, "优惠券不存在或不可用")
			return
		}

		// 校验有效期（已在查询中判断，但双重保险）
		now := time.Now()
		if now.Before(userCoupon.ValidStart) || now.After(userCoupon.ValidEnd) {
			utils.Fail(c, "优惠券已过期")
			return
		}

		coupon := userCoupon.Coupon

		// 商品范围校验
		scopeOK := false
		if coupon.UseScope == models.UseScopeAll {
			scopeOK = true
		} else if coupon.UseScope == models.UseScopeSpec {
			productIDs := strings.Split(coupon.ProductIds, ",")
			for _, pid := range productIDs {
				if strings.TrimSpace(pid) == req.ProductID {
					scopeOK = true
					break
				}
			}
		}
		if !scopeOK {
			utils.Fail(c, "优惠券不适用于当前商品")
			return
		}

		// 门槛校验
		if orderTotal < coupon.FullAmount {
			utils.Fail(c, fmt.Sprintf("订单金额未满 %.2f 元，无法使用该优惠券", coupon.FullAmount))
			return
		}

		// 计算优惠金额
		switch coupon.Type {
		case models.CouponTypeFullReduce: // 满减券
			discountAmount = coupon.ReduceAmount
		case models.CouponTypeNoThreshold: // 无门槛券
			discountAmount = coupon.ReduceAmount
		case models.CouponTypeDiscount: // 折扣券
			discountAmount = orderTotal * (1 - coupon.DiscountRate)
			if coupon.MaxReduce > 0 && discountAmount > coupon.MaxReduce {
				discountAmount = coupon.MaxReduce
			}
		default:
			utils.Fail(c, "不支持的优惠券类型")
			return
		}

		if discountAmount > orderTotal {
			discountAmount = orderTotal
		}

		usedUserCouponID = userCoupon.ID.Int64()
	}

	// ---------- 5. 计算最终实付金额 ----------
	actualAmount := orderTotal - discountAmount
	if actualAmount < 0 {
		actualAmount = 0
	}

	// ---------- 6. 创建订单数据 ----------
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())

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
	if usedUserCouponID != 0 {
		order.CouponID = utils.Int64Str(usedUserCouponID)
	}

	// 地址快照
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

	// ---------- 7. 事务执行 ----------
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 创建订单
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		orderAddress.OrderID = order.ID
		if err := tx.Create(&orderAddress).Error; err != nil {
			return err
		}
		// 扣减库存、生成订单项
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
		// 更新用户优惠券状态
		if usedUserCouponID != 0 {
			result := tx.Model(&models.UserCoupon{}).
				Where("id = ? AND user_id = ? AND status = ?", usedUserCouponID, userID, models.UserCouponUnused).
				Updates(map[string]interface{}{
					"status":   models.UserCouponUsed,
					"order_id": order.ID,
					"used_at":  time.Now(),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("优惠券已被使用或不存在")
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
