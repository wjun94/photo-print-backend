package admin

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateCouponReq 创建优惠券请求
type CreateCouponReq struct {
	Name           string     `json:"name" binding:"required"`
	Type           int        `json:"type" binding:"required,min=1,max=3"`
	FullAmount     float64    `json:"fullAmount"`
	ReduceAmount   float64    `json:"reduceAmount"`
	DiscountRate   float64    `json:"discountRate"`
	MaxReduce      float64    `json:"maxReduce"`
	UseScope       int        `json:"useScope" binding:"required,min=1,max=2"`
	ProductIds     string     `json:"productIds"`
	TimeType       int        `json:"timeType" binding:"required,min=1,max=2"`
	ValidStart     *time.Time `json:"validStart"`
	ValidEnd       *time.Time `json:"validEnd"`
	ValidDays      int        `json:"validDays"`
	TotalStock     int        `json:"totalStock"`
	UserLimitType  int        `json:"userLimitType" binding:"required,min=1,max=3"`
	UserLimitNum   int        `json:"userLimitNum"`
	TargetUserType int        `json:"targetUserType"`
	ReceiveStart   time.Time  `json:"receiveStart" binding:"required"`
	ReceiveEnd     time.Time  `json:"receiveEnd" binding:"required"`
	Desc           string     `json:"desc"`
	// 以下字段有默认值，不要求前端必传
	PublishStatus int `json:"publishStatus"` // 默认0（未发布）
	// 折扣券默认折扣率1.0
}

// CreateCoupon 创建优惠券
// @Summary 创建优惠券
// @Tags 优惠券管理
// @Accept json
// @Produce json
// @Param coupon body CreateCouponReq true "优惠券信息"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/coupon [post]
func CreateCoupon(c *gin.Context) {
	var req CreateCouponReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.PublishStatus == 0 {
		req.PublishStatus = 0 // 默认未发布
	}
	if req.DiscountRate == 0 {
		req.DiscountRate = 1.0
	}
	if req.UserLimitNum == 0 && req.UserLimitType == 2 {
		req.UserLimitNum = 1
	}

	// 校验优惠券类型相关字段
	switch req.Type {
	case 1: // 满减券
		if req.FullAmount <= 0 || req.ReduceAmount <= 0 {
			utils.Fail(c, "满减券需要填写满减门槛和减免金额")
			return
		}
	case 2: // 无门槛券
		if req.ReduceAmount <= 0 {
			utils.Fail(c, "无门槛券需要填写减免金额")
			return
		}
	case 3: // 折扣券
		if req.DiscountRate <= 0 || req.DiscountRate > 1 {
			utils.Fail(c, "折扣率需在0-1之间")
			return
		}
	}
	// 有效期校验
	if req.TimeType == 1 {
		if req.ValidStart == nil || req.ValidEnd == nil {
			utils.Fail(c, "固定有效期需要填写起始和结束时间")
			return
		}
	} else if req.TimeType == 2 {
		if req.ValidDays <= 0 {
			utils.Fail(c, "领券后有效天数必须大于0")
			return
		}
	}
	// 领券时间校验
	if req.ReceiveStart.After(req.ReceiveEnd) {
		utils.Fail(c, "领券开始时间不能晚于结束时间")
		return
	}

	// 构造优惠券对象
	coupon := models.Coupon{
		Name:           req.Name,
		Type:           models.CouponType(req.Type),
		PublishStatus:  models.PublishStatus(req.PublishStatus),
		FullAmount:     req.FullAmount,
		ReduceAmount:   req.ReduceAmount,
		DiscountRate:   req.DiscountRate,
		MaxReduce:      req.MaxReduce,
		UseScope:       models.UseScope(req.UseScope),
		ProductIds:     req.ProductIds,
		TimeType:       models.TimeType(req.TimeType),
		ValidStart:     req.ValidStart,
		ValidEnd:       req.ValidEnd,
		ValidDays:      req.ValidDays,
		TotalStock:     req.TotalStock,
		ReceivedNum:    0,
		UserLimitType:  req.UserLimitType,
		UserLimitNum:   req.UserLimitNum,
		TargetUserType: req.TargetUserType,
		ReceiveStart:   req.ReceiveStart,
		ReceiveEnd:     req.ReceiveEnd,
		Desc:           req.Desc,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := database.DB.Create(&coupon).Error; err != nil {
		utils.Fail(c, "创建失败: "+err.Error())
		return
	}
	utils.Success(c, coupon)
}

// GetCouponList 优惠券列表（分页、筛选）
// @Summary 优惠券列表
// @Tags 优惠券管理(后台)
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query int false "发布状态 0-未发布 1-已发布 2-下架"
// @Param name query string false "券名称模糊搜索"
// @Success 200 {object} utils.Response{data=object{list=[]models.Coupon,total=int64,page=int,size=int}}
// @Router /api/v1/admin/coupons [get]
func GetCouponList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	statusStr := c.Query("status")
	name := c.Query("name")
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size

	var coupons []models.Coupon
	var total int64
	query := database.DB.Model(&models.Coupon{})
	if statusStr != "" {
		query = query.Where("publish_status = ?", statusStr)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&coupons)
	utils.Success(c, gin.H{
		"list":  coupons,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetCouponDetail 优惠券详情
// @Summary 优惠券详情
// @Tags 优惠券管理(后台)
// @Produce json
// @Param id path int true "优惠券ID"
// @Success 200 {object} utils.Response{data=models.Coupon}
// @Router /api/v1/admin/coupons/{id} [get]
func GetCouponDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		utils.Fail(c, "优惠券不存在")
		return
	}
	utils.Success(c, coupon)
}

// UpdateCouponReq 更新优惠券请求
type UpdateCouponReq struct {
	Name           *string  `json:"name"`
	Type           *int     `json:"type"`
	PublishStatus  *int     `json:"publishStatus"`
	FullAmount     *float64 `json:"fullAmount"`
	ReduceAmount   *float64 `json:"reduceAmount"`
	DiscountRate   *float64 `json:"discountRate"`
	MaxReduce      *float64 `json:"maxReduce"`
	UseScope       *int     `json:"useScope"`
	ProductIds     *string  `json:"productIds"`
	TimeType       *int     `json:"timeType"`
	ValidStart     *string  `json:"validStart"`
	ValidEnd       *string  `json:"validEnd"`
	ValidDays      *int     `json:"validDays"`
	TotalStock     *int     `json:"totalStock"`
	UserLimitType  *int     `json:"userLimitType"`
	UserLimitNum   *int     `json:"userLimitNum"`
	TargetUserType *int     `json:"targetUserType"`
	ReceiveStart   *string  `json:"receiveStart"`
	ReceiveEnd     *string  `json:"receiveEnd"`
	Desc           *string  `json:"desc"`
}

// UpdateCoupon 更新优惠券
// @Summary 更新优惠券
// @Tags 优惠券管理(后台)
// @Accept json
// @Produce json
// @Param id path int true "优惠券ID"
// @Param coupon body UpdateCouponReq true "待更新字段"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/coupons/{id} [put]
func UpdateCoupon(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var coupon models.Coupon
	if err := database.DB.First(&coupon, id).Error; err != nil {
		utils.Fail(c, "优惠券不存在")
		return
	}
	var req UpdateCouponReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.PublishStatus != nil {
		updates["publish_status"] = *req.PublishStatus
	}
	if req.FullAmount != nil {
		updates["full_amount"] = *req.FullAmount
	}
	if req.ReduceAmount != nil {
		updates["reduce_amount"] = *req.ReduceAmount
	}
	if req.DiscountRate != nil {
		updates["discount_rate"] = *req.DiscountRate
	}
	if req.MaxReduce != nil {
		updates["max_reduce"] = *req.MaxReduce
	}
	if req.UseScope != nil {
		updates["use_scope"] = *req.UseScope
	}
	if req.ProductIds != nil {
		updates["product_ids"] = *req.ProductIds
	}
	if req.TimeType != nil {
		updates["time_type"] = *req.TimeType
	}
	// 处理时间字段
	if req.ValidStart != nil {
		if *req.ValidStart == "" {
			updates["valid_start"] = nil
		} else {
			t, err := time.Parse("2006-01-02 15:04:05", *req.ValidStart)
			if err != nil {
				utils.Fail(c, "validStart 格式错误")
				return
			}
			updates["valid_start"] = t
		}
	}
	if req.ValidEnd != nil {
		if *req.ValidEnd == "" {
			updates["valid_end"] = nil
		} else {
			t, err := time.Parse("2006-01-02 15:04:05", *req.ValidEnd)
			if err != nil {
				utils.Fail(c, "validEnd 格式错误")
				return
			}
			updates["valid_end"] = t
		}
	}
	if req.ValidDays != nil {
		updates["valid_days"] = *req.ValidDays
	}
	if req.TotalStock != nil {
		updates["total_stock"] = *req.TotalStock
	}
	if req.UserLimitType != nil {
		updates["user_limit_type"] = *req.UserLimitType
	}
	if req.UserLimitNum != nil {
		updates["user_limit_num"] = *req.UserLimitNum
	}
	if req.TargetUserType != nil {
		updates["target_user_type"] = *req.TargetUserType
	}
	if req.ReceiveStart != nil {
		fmt.Println("-----")
		fmt.Println(*req.ReceiveStart)
		t, err := time.Parse(time.RFC3339, *req.ReceiveStart)
		fmt.Println(t)

		if err != nil {
			utils.Fail(c, "receiveStart 格式错误")
			return
		}
		updates["receive_start"] = t
	}
	if req.ReceiveEnd != nil {
		t, err := time.Parse(time.RFC3339, *req.ReceiveEnd)
		if err != nil {
			utils.Fail(c, "receiveEnd 格式错误")
			return
		}
		updates["receive_end"] = t
	}
	if req.Desc != nil {
		updates["desc"] = *req.Desc
	}
	updates["updated_at"] = time.Now()
	if err := database.DB.Model(&coupon).Updates(updates).Error; err != nil {
		utils.Fail(c, "更新失败")
		return
	}
	utils.Success(c, nil)
}

// DeleteCoupon 删除优惠券（物理删除，需谨慎）
// @Summary 删除优惠券
// @Tags 优惠券管理(后台)
// @Param id path int true "优惠券ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/coupons/{id} [delete]
func DeleteCoupon(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	// 检查是否已有用户领取（防止删除已发放的优惠券）
	var count int64
	database.DB.Model(&models.UserCoupon{}).Where("coupon_id = ?", id).Count(&count)
	if count > 0 {
		utils.Fail(c, "已有用户领取该优惠券，无法删除")
		return
	}
	if err := database.DB.Delete(&models.Coupon{}, id).Error; err != nil {
		utils.Fail(c, "删除失败")
		return
	}
	utils.Success(c, nil)
}
