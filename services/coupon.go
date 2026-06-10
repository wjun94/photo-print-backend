package services

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"time"
)

// ValidateCoupon 校验优惠券可用性，返回优惠金额和优惠券模板
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
	if coupon.UseScope == models.UseScopeSpec && coupon.ProductIDs != "" {
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
