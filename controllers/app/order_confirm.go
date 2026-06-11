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

	// ---------- 验证地址 ----------
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

	// ---------- 预检商品规格 ----------
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

	// ---------- 运费与初始金额 ----------
	freight := utils.CalculateFreight(totalAmount)
	actualAmount := totalAmount + freight
	discountAmount := 0.0
	var usedUserCouponID int64 // 用户优惠券实例ID

	// ---------- 优惠券处理（修正） ----------
	if req.CouponID != "" {
		// 打印前端传入的原始值，用于调试
		fmt.Printf("前端传入的 couponId: %s\n", req.CouponID)

		userCouponID, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err != nil {
			utils.Fail(c, "优惠券ID格式错误")
			return
		}

		// 直接查询用户优惠券实例，并预加载模板信息
		var userCoupon models.UserCoupon
		err = database.DB.Preload("Coupon").Where("coupon_id = ? AND user_id = ? AND status = ?",
			userCouponID, userID, models.UserCouponUnused).First(&userCoupon).Error
		if err != nil {
			utils.Fail(c, "优惠券不存在或不可用")
			return
		}

		// 检查有效期
		now := time.Now()
		if now.Before(userCoupon.ValidStart) || now.After(userCoupon.ValidEnd) {
			utils.Fail(c, "优惠券已过期")
			return
		}

		coupon := userCoupon.Coupon
		orderTotal := totalAmount + freight

		// 检查商品适用范围
		scopeOK := false
		if coupon.UseScope == models.UseScopeAll {
			scopeOK = true
		} else if coupon.UseScope == models.UseScopeSpec {
			productIDs := strings.Split(coupon.ProductIDs, ",")
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

		// 门槛检查
		if orderTotal < coupon.FullAmount {
			utils.Fail(c, fmt.Sprintf("订单金额未满 %.2f 元，无法使用该优惠券", coupon.FullAmount))
			return
		}

		// 计算抵扣金额
		if coupon.Type == 1 { // 满减
			discountAmount = coupon.ReduceAmount
			if discountAmount > orderTotal {
				discountAmount = orderTotal
			}
		} else if coupon.Type == 2 { // 折扣
			discountAmount = orderTotal * (1 - coupon.DiscountRate)
			if coupon.MaxReduce > 0 && discountAmount > coupon.MaxReduce {
				discountAmount = coupon.MaxReduce
			}
			if discountAmount > orderTotal {
				discountAmount = orderTotal
			}
		} else {
			utils.Fail(c, "不支持的优惠券类型")
			return
		}

		actualAmount = orderTotal - discountAmount
		if actualAmount < 0 {
			actualAmount = 0
		}
		usedUserCouponID = userCoupon.ID.Int64()
	}

	// ---------- 生成订单号 ----------
	orderNo := fmt.Sprintf("PO%d", time.Now().UnixNano())

	// ---------- 构建订单对象 ----------
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
	fmt.Println("----------")
	fmt.Println(usedUserCouponID)
	fmt.Println(req.CouponID)
	if usedUserCouponID != 0 {
		order.CouponID = utils.Int64Str(usedUserCouponID) // 存储用户优惠券实例ID
	}

	// 订单地址快照（略，同原代码）
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

	// ---------- 事务执行 ----------
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
		fmt.Println("----------2")
		fmt.Println(usedUserCouponID)
		// 更新用户优惠券状态（使用用户优惠券实例ID）
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
