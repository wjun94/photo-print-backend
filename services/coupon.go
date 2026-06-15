package services

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"strings"
	"time"
)

// ValidateCoupon 校验优惠券可用性，返回优惠金额和优惠券模板(提交订单使用)
func ValidateCoupon(couponID int64, userID int64, totalAmount float64) (discount float64, coupon models.Coupon, err error) {
	var userCoupon models.UserCoupon
	if err := database.DB.Where("user_id = ? AND coupon_id = ? AND status = ?", userID, couponID, models.UserCouponUnused).First(&userCoupon).Error; err != nil {
		return 0, coupon, fmt.Errorf("优惠券不存在或已使用")
	}
	now := time.Now()
	if now.Before(userCoupon.ValidStart) || now.After(userCoupon.ValidEnd) {
		return 0, coupon, fmt.Errorf("优惠券不在有效期内")
	}
	if err := database.DB.First(&coupon, userCoupon.CouponID).Error; err != nil {
		return 0, coupon, fmt.Errorf("优惠券模板无效")
	}
	// 使用范围校验（示例简化，可根据业务扩展）
	if coupon.UseScope == models.UseScopeSpec && coupon.ProductIds != "" {
		// 可在此处校验订单中的商品是否都在指定商品列表中
		// 此处省略
	}
	switch coupon.Type {
	case models.CouponTypeFullReduce:
		if totalAmount < coupon.FullAmount {
			return 0, coupon, fmt.Errorf("未达到满减门槛 %.2f", coupon.FullAmount)
		}
		discount = coupon.ReduceAmount
	case models.CouponTypeNoThreshold:
		discount = coupon.ReduceAmount
	case models.CouponTypeDiscount:
		discount = totalAmount * (1 - coupon.DiscountRate)
		if coupon.MaxReduce > 0 && discount > coupon.MaxReduce {
			discount = coupon.MaxReduce
		}
	default:
		return 0, coupon, fmt.Errorf("无效优惠券类型")
	}
	if discount > totalAmount {
		discount = totalAmount
	}
	return discount, coupon, nil
}

// ------------------------ 预览订单使用 ----------------------
// ValidateUserCoupon 校验用户优惠券是否适用于当前订单
// userCoupon: 用户已领取的优惠券记录
// totalAmount: 订单商品总额 + 运费
// productIDs: 订单中所有商品的 ID 列表（用于指定商品限制）
// 返回优惠金额和错误信息
func ValidateUserCoupon(userCoupon models.UserCoupon, totalAmount float64, productIDs []string) (float64, error) {
	now := time.Now()
	if userCoupon.Status != models.UserCouponUnused {
		return 0, fmt.Errorf("优惠券已使用或已失效")
	}
	if now.Before(userCoupon.ValidStart) || now.After(userCoupon.ValidEnd) {
		return 0, fmt.Errorf("优惠券不在有效期内")
	}

	var coupon models.Coupon
	if err := database.DB.First(&coupon, userCoupon.CouponID).Error; err != nil {
		return 0, fmt.Errorf("优惠券模板不存在")
	}

	// 适用商品范围检查（use_scope=2 指定商品）
	if coupon.UseScope == models.UseScopeSpec {
		if len(productIDs) == 0 {
			return 0, fmt.Errorf("订单无商品，无法使用该优惠券")
		}
		allowed := strings.Split(coupon.ProductIds, ",")
		allowedMap := make(map[string]bool)
		for _, id := range allowed {
			allowedMap[id] = true
		}
		for _, pid := range productIDs {
			if !allowedMap[pid] {
				return 0, fmt.Errorf("优惠券不适用于订单中的部分商品")
			}
		}
	}

	var discount float64
	switch coupon.Type {
	case models.CouponTypeFullReduce:
		if totalAmount < coupon.FullAmount {
			return 0, fmt.Errorf("未达到满减门槛 %.2f", coupon.FullAmount)
		}
		discount = coupon.ReduceAmount
	case models.CouponTypeNoThreshold:
		discount = coupon.ReduceAmount
	case models.CouponTypeDiscount:
		discount = totalAmount * (1 - coupon.DiscountRate)
		if coupon.MaxReduce > 0 && discount > coupon.MaxReduce {
			discount = coupon.MaxReduce
		}
	default:
		return 0, fmt.Errorf("无效优惠券类型")
	}
	if discount > totalAmount {
		discount = totalAmount
	}
	return discount, nil
}

// GetBestCouponForOrder 为订单选择最佳优惠券（优惠金额最大）
// 返回优惠金额、优惠券ID（字符串）及错误
func GetBestCouponForOrder(userID int64, totalAmount float64, productIDs []string) (discount float64, couponIDStr string, err error) {
	var userCoupons []models.UserCoupon
	if err := database.DB.
		Where("user_id = ? AND status = ?", userID, models.UserCouponUnused).
		Preload("Coupon").
		Find(&userCoupons).Error; err != nil {
		return 0, "", err
	}

	var bestDiscount float64
	var bestUserCoupon *models.UserCoupon
	for i, uc := range userCoupons {
		disc, err := ValidateUserCoupon(uc, totalAmount, productIDs)
		if err == nil && disc > bestDiscount {
			bestDiscount = disc
			bestUserCoupon = &userCoupons[i]
		}
	}
	if bestUserCoupon != nil {
		return bestDiscount, bestUserCoupon.CouponID.String(), nil
	}
	return 0, "", nil
}

// ------------------------ 预览订单使用 end ----------------------
