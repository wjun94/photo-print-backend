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
			if !strings.Contains(","+coupon.ProductIds+",", ","+req.ProductID+",") {
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

// GetMyCoupons 我的优惠券（自动返回未使用券，并按是否过期动态设置状态）
// @Summary 我的优惠券
// @Tags 优惠券
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页条数" default(10)
// @Success 200 {object} utils.Response{data=object{list=[]models.UserCoupon,total=int,page=int,pageSize=int}}
// @Router /api/v1/wx/coupon/list [get]
func GetMyCoupons(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	// 1. 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var userCoupons []models.UserCoupon
	var total int64

	// 2. 构建基础查询（不包含分页）
	baseQuery := database.DB.Model(&models.UserCoupon{}).
		Where("user_id = ? AND status IN (?) AND valid_end >= ?", userID, []int{0, 2}, sevenDaysAgo)

	// 3. 获取总数（用于前端分页）
	if err := baseQuery.Count(&total).Error; err != nil {
		utils.Fail(c, "查询总数失败")
		return
	}

	// 4. 执行分页查询（预加载关联的 Coupon，并按领取时间倒序）
	err := baseQuery.
		Preload("Coupon").
		Order("received_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&userCoupons).Error

	if err != nil {
		utils.Fail(c, "查询失败")
		return
	}

	// 5. 动态设置状态（过期状态在内存中计算）
	for i := range userCoupons {
		if userCoupons[i].ValidEnd.Before(now) {
			userCoupons[i].Status = models.UserCouponExpired // 2
		} else {
			userCoupons[i].Status = models.UserCouponUnused // 0
		}
	}

	// 6. 返回分页数据（包含列表、总数、当前页码、每页大小）
	utils.Success(c, gin.H{
		"list":     userCoupons,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CouponDetail 优惠券详情（含用户领取状态）
type CouponDetail struct {
	models.Coupon
	IsReceived   bool  `json:"isReceived"`   // 历史是否领过
	RemainCanGet int64 `json:"remainCanGet"` // 还可领取次数（累计剩余）
	Status       int   `json:"status"`       // 0-可领取 1-可使用 2-已达上限
}

// GetProductCoupons 获取商品可领取的优惠券列表（用户维度累计限制）
// @Summary 获取商品可领优惠券
// @Description 根据商品ID，返回当前可领取且适用于该商品的优惠券列表，并附带用户累计领取总数和剩余可领次数（基于累计次数限制）。
// @Description 注意：用户使用/过期后仍会计入已领取总数，因此每人最多领取次数受UserLimitType/UserLimitNum限制（累计制）。
// @Tags 优惠券
// @Accept json
// @Produce json
// @Param productId path string true "商品ID"
// @Success 200 {object} utils.Response{data=[]CouponDetail}
// @Router /api/v1/wx/coupon/product/{productId} [get]
func GetProductCoupons(c *gin.Context) {
	userID, _ := utils.GetUserID(c)
	productIdStr := c.Param("productId")
	now := time.Now()

	var coupons []models.Coupon
	query := database.DB.Where("publish_status = ?", models.PublishStatusPublished).
		Where("receive_start <= ? AND receive_end >= ?", now, now).
		Where("use_scope = ? OR (use_scope = ? AND find_in_set(?, product_ids) > 0)",
			models.UseScopeAll, models.UseScopeSpec, productIdStr)

	if err := query.Find(&coupons).Error; err != nil {
		utils.Fail(c, "查询优惠券失败")
		return
	}

	// 累计领取总数 map
	totalMap := make(map[utils.Int64Str]int64)
	// 有效持有数量 map（未使用且未过期）
	validMap := make(map[utils.Int64Str]int64)
	if userID != 0 {
		// 累计总数
		type CountResult struct {
			CouponID utils.Int64Str
			Cnt      int64
		}
		var totals []CountResult
		database.DB.Model(&models.UserCoupon{}).
			Where("user_id = ?", userID).
			Select("coupon_id, count(*) as cnt").
			Group("coupon_id").
			Scan(&totals)
		for _, t := range totals {
			totalMap[t.CouponID] = t.Cnt
		}

		// 有效持有数
		var valids []CountResult
		database.DB.Model(&models.UserCoupon{}).
			Where("user_id = ? AND status = ? AND valid_start <= ? AND valid_end >= ?",
				userID, models.UserCouponUnused, now, now).
			Select("coupon_id, count(*) as cnt").
			Group("coupon_id").
			Scan(&valids)
		for _, v := range valids {
			validMap[v.CouponID] = v.Cnt
		}
		// 获取每个优惠券模板下用户持有的第一张有效券ID
		type FirstValid struct {
			CouponID utils.Int64Str
			ID       utils.Int64Str
		}
	}

	result := make([]CouponDetail, 0)

	for _, coupon := range coupons {
		// 库存检查
		if coupon.TotalStock > 0 && coupon.ReceivedNum >= coupon.TotalStock {
			continue
		}

		receivedTotal := totalMap[coupon.ID]
		validCnt := validMap[coupon.ID]
		isReceived := receivedTotal > 0

		// 计算累计剩余可领次数
		var remain int64 = -1
		switch coupon.UserLimitType {
		case 1:
			remain = -1
		case 2:
			remain = int64(coupon.UserLimitNum) - receivedTotal
			if remain < 0 {
				remain = 0
			}
		case 3:
			if receivedTotal >= 1 {
				remain = 0
			} else {
				remain = 1
			}
		}

		// 状态判断  0-可领取 1-可使用 2-已达上限
		var status int
		if validCnt > 0 {
			// 有可用的券
			status = 1 // 可使用
		} else {
			if remain > 0 || remain == -1 {
				status = 0 // 可领取
			} else {
				status = 2 // 已达上限（已使用完或累计已满）
			}
		}

		result = append(result, CouponDetail{
			Coupon:       coupon,
			IsReceived:   isReceived,
			RemainCanGet: remain,
			Status:       status,
		})
	}

	utils.Success(c, result)
}
