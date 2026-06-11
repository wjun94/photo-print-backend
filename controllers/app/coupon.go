package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReceiveCouponReq struct {
	CouponID  string `json:"couponId" binding:"required"`
	ProductID string `json:"productId"` // 可选，若提供则校验该商品是否在适用范围内
}

// ReceiveCoupon 用户领取优惠券
// @Summary 领取优惠券
// @Tags 优惠券
// @Param request body ReceiveCouponReq true "优惠券ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/wx/coupon/receive [post]
func ReceiveCoupon(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req ReceiveCouponReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}
	couponID, err := strconv.ParseInt(req.CouponID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效优惠券ID")
		return
	}

	var coupon models.Coupon
	if err := database.DB.First(&coupon, couponID).Error; err != nil {
		utils.Fail(c, "优惠券不存在")
		return
	}

	// 校验领取状态
	if coupon.PublishStatus != models.PublishStatusPublished {
		utils.Fail(c, "优惠券未发布或已下架")
		return
	}
	now := time.Now()
	if now.Before(coupon.ReceiveStart) || now.After(coupon.ReceiveEnd) {
		utils.Fail(c, "不在领券时间范围内")
		return
	}
	if coupon.TotalStock > 0 && coupon.ReceivedNum >= coupon.TotalStock {
		utils.Fail(c, "优惠券已领完")
		return
	}

	// 增加商品适用性校验
	if req.ProductID != "" {
		var product models.Product
		if err := database.DB.First(&product, req.ProductID).Error; err != nil {
			utils.Fail(c, "商品不存在")
			return
		}
		if coupon.UseScope == models.UseScopeSpec {
			if !strings.Contains(","+coupon.ProductIDs+",", ","+req.ProductID+",") {
				utils.Fail(c, "该优惠券不适用于当前商品")
				return
			}
		}
	}

	// 用户领取数量限制
	var userCount int64
	database.DB.Model(&models.UserCoupon{}).Where("user_id = ? AND coupon_id = ?", userID, couponID).Count(&userCount)
	if coupon.UserLimitType == 2 && userCount >= int64(coupon.UserLimitNum) {
		utils.Fail(c, "您已达到领取上限")
	} else if coupon.UserLimitType == 3 && userCount >= 1 {
		utils.Fail(c, "您已领取过该优惠券")
	}

	// 新老用户限制（简单示例，可根据注册时间判断）
	// 此处略去，实际可查询用户注册时间

	// 计算用户优惠券有效期
	var validStart, validEnd time.Time
	if coupon.TimeType == models.TimeTypeFixed {
		if coupon.ValidStart == nil || coupon.ValidEnd == nil {
			utils.Fail(c, "优惠券有效期配置错误")
			return
		}
		validStart = *coupon.ValidStart
		validEnd = *coupon.ValidEnd
	} else {
		validStart = now
		validEnd = now.AddDate(0, 0, coupon.ValidDays)
	}

	// 开启事务: 增加领取数，插入用户优惠券
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 更新已领取数量（行锁）
		if err := tx.Model(&coupon).Where("id = ? AND received_num < total_stock", couponID).
			Update("received_num", gorm.Expr("received_num + 1")).Error; err != nil {
			return err
		}
		userCoupon := models.UserCoupon{
			UserID:     utils.Int64Str(userID),
			CouponID:   utils.Int64Str(couponID),
			Status:     models.UserCouponUnused,
			ReceivedAt: now,
			ValidStart: validStart,
			ValidEnd:   validEnd,
		}
		if err := tx.Create(&userCoupon).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "领取失败，请稍后重试")
		return
	}
	utils.Success(c, nil)
}

// GetMyCoupons 获取当前用户可用的优惠券
// @Summary 我的优惠券
// @Tags 优惠券
// @Param status query int false "状态 0未使用 1已使用 2已过期" default(0)
// @Success 200 {object} utils.Response{data=[]models.UserCoupon}
// @Router /api/v1/wx/coupon/list [get]
func GetMyCoupons(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	statusStr := c.DefaultQuery("status", "0")
	status, _ := strconv.Atoi(statusStr)

	var userCoupons []models.UserCoupon
	now := time.Now()
	// 先查出用户优惠券，然后过滤已过期的状态
	err := database.DB.Where("user_id = ? AND status = ?", userID, status).Find(&userCoupons).Error
	if err != nil {
		utils.Fail(c, "查询失败")
		return
	}

	// 对于未使用的优惠券，自动将过期状态更新（非持久，前端显示用）
	if status == 0 {
		for i, uc := range userCoupons {
			if uc.ValidEnd.Before(now) && uc.Status == models.UserCouponUnused {
				userCoupons[i].Status = models.UserCouponExpired
			}
		}
	}
	utils.Success(c, userCoupons)
}

// GetProductCoupons 获取商品可领取的优惠券列表
// @Summary 获取商品可领优惠券
// @Description 根据商品ID，返回当前可领取且适用于该商品的优惠券列表
// @Tags 优惠券
// @Accept json
// @Produce json
// @Param productId path string true "商品ID"
// @Success 200 {object} utils.Response{data=[]models.Coupon}
// @Router /api/v1/wx/coupon/product/{productId} [get]
func GetProductCoupons(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	productIdStr := c.Param("productId")
	now := time.Now()

	var coupons []models.Coupon
	query := database.DB.Where("publish_status = ?", models.PublishStatusPublished).
		Where("receive_start <= ? AND receive_end >= ?", now, now)
	query = query.Where("use_scope = ? OR (use_scope = ? AND find_in_set(?, product_ids) > 0)",
		models.UseScopeAll, models.UseScopeSpec, productIdStr)
	if err := query.Find(&coupons).Error; err != nil {
		utils.Fail(c, "查询优惠券失败")
		return
	}

	// 1. 批量获取当前用户对每张优惠券的已领取数量
	receivedCountMap := make(map[utils.Int64Str]int64)
	if userID != 0 {
		// 临时结构体，用于接收 group by 结果
		type UserCouponCount struct {
			CouponID utils.Int64Str
			Cnt      int64
		}
		var counts []UserCouponCount
		err := database.DB.Model(&models.UserCoupon{}).
			Where("user_id = ?", userID).
			Select("coupon_id, count(*) as cnt").
			Group("coupon_id").
			Scan(&counts).Error
		if err == nil {
			for _, c := range counts {
				receivedCountMap[c.CouponID] = c.Cnt
			}
		}
	}

	// 2. 构造返回结果，附加领取状态和剩余可领次数
	type CouponDetail struct {
		models.Coupon
		IsReceived   bool  `json:"isReceived"`   // 是否已领取（至少一张）
		RemainCanGet int64 `json:"remainCanGet"` // 还可领取张数，-1表示不限制
	}
	result := make([]CouponDetail, 0)

	for _, coupon := range coupons {
		// 库存检查：总库存有限且已发数量已达上限时不展示
		if coupon.TotalStock > 0 && coupon.ReceivedNum >= coupon.TotalStock {
			continue
		}

		// 获取当前用户的已领数量
		var receivedCnt int64 = 0
		if userID != 0 {
			receivedCnt = receivedCountMap[coupon.ID]
		}
		isReceived := receivedCnt > 0

		// 计算剩余可领次数（用户维度）
		var remain int64 = -1 // 默认无限制
		switch coupon.UserLimitType {
		case 1: // 无限
			remain = -1
		case 2: // 限总量
			remain = int64(coupon.UserLimitNum) - receivedCnt
			if remain < 0 {
				remain = 0
			}
		case 3: // 限1张
			if receivedCnt >= 1 {
				remain = 0
			} else {
				remain = 1
			}
		}

		result = append(result, CouponDetail{
			Coupon:       coupon,
			IsReceived:   isReceived,
			RemainCanGet: remain,
		})
	}

	utils.Success(c, result)
}
