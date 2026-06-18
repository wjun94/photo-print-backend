package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/services"
	"photo-print-backend/utils"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ========================
// 预览订单相关结构体
// ========================

// PreviewOrderItem 预览订单项（用户上传的图片和数量）
type PreviewOrderItem struct {
	ImageURL string `json:"imageUrl"`                          // 图片URL（action=upload时必填）
	Quantity int    `json:"quantity" binding:"required,min=1"` // 数量
}

// PreviewOrderReq 预览订单请求
type PreviewOrderReq struct {
	ProductID string             `json:"productId" binding:"required"`   // 商品ID
	SpecID    string             `json:"specId" binding:"required"`      // 规格ID
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"` // 订单项列表
	CouponID  string             `json:"couponId"`                       // 用户优惠券实例ID（可选）
}

// PreviewItemResponse 预览订单项响应
type PreviewItemResponse struct {
	ProductID   string  `json:"productId"`   // 商品ID
	ProductName string  `json:"productName"` // 商品名称
	SpecID      string  `json:"specId"`      // 规格ID
	SpecName    string  `json:"specName"`    // 规格名称
	Price       float64 `json:"price"`       // 单价
	Quantity    int     `json:"quantity"`    // 数量
	Subtotal    float64 `json:"subtotal"`    // 小计
	ImageURL    string  `json:"imageUrl"`    // 图片URL
}

// PreviewCouponResponse 预览优惠券响应
type PreviewCouponResponse struct {
	UserCouponID string  `json:"userCouponId"` // 用户优惠券实例ID
	ID           string  `json:"id"`           // 优惠券模板ID
	Name         string  `json:"name"`         // 优惠券名称
	FullAmount   float64 `json:"fullAmount"`   // 满减门槛金额
	ReduceAmount float64 `json:"reduceAmount"` // 减额（满减/无门槛）
	DiscountRate float64 `json:"discountRate"` // 折扣率（0.8=8折）
	MaxReduce    float64 `json:"maxReduce"`    // 折扣最高减额
	Type         int     `json:"type"`         // 类型：1满减 2无门槛 3折扣
	AmountDesc   string  `json:"amountDesc"`   // 优惠文案，如“减10元”
	MinAmount    float64 `json:"minAmount"`    // 最低消费门槛
	ValidStart   string  `json:"validStart"`   // 有效期开始
	ValidEnd     string  `json:"validEnd"`     // 有效期结束
	Status       int     `json:"status"`       // 1-可用 0-不可用
	Reason       string  `json:"reason"`       // 不可用原因
	Discount     float64 `json:"discount"`     // 该券在当前订单可减免的金额
}

// PreviewOrderResponseData 订单预览响应数据
type PreviewOrderResponseData struct {
	Items            []PreviewItemResponse        `json:"items"`
	Specs            []models.SpecSummaryResponse `json:"specs"`
	Coupons          []PreviewCouponResponse      `json:"coupons"`
	TotalAmount      float64                      `json:"totalAmount"`
	Freight          float64                      `json:"freight"`
	DiscountAmount   float64                      `json:"discountAmount"`
	ActualAmount     float64                      `json:"actualAmount"`
	DefaultAddress   interface{}                  `json:"defaultAddress"` // 或定义为 AddressResponse 结构体
	SelectedCouponId string                       `json:"selectedCouponId"`
}

// PreviewOrder 确认订单页面预览（支持优惠券自动选最佳）
// @Summary 订单预览
// @Description 计算商品总金额、运费，返回可用的优惠券列表（含每张券可优惠金额），自动选择最佳券或使用用户指定券
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body PreviewOrderReq true "预览请求"
// @Success 200 {object} utils.Response{data=PreviewOrderResponseData} "成功返回预览数据"
// @Router /api/v1/wx/order/preview [post]
func PreviewOrder(c *gin.Context) {
	// 获取当前登录用户ID
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

	// ---------- 1. 商品校验与金额计算 ----------
	var previewItems []PreviewItemResponse
	var totalAmount float64
	specSummaryMap := make(map[string]*models.SpecSummaryResponse)

	// 遍历每个订单项（目前仅支持单商品多图片，但保留数组结构）
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

		// 查询规格并预加载商品信息
		var spec models.Spec
		if err := database.DB.First(&spec, specID).Error; err != nil {
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
			utils.Fail(c, "规格库存不足")
			return
		}

		// 处理图片：action=upload必须由用户上传，否则使用商品封面
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

		// 规格汇总（按规格ID聚合，用于订单卡片展示）
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

	// 将规格汇总从 map 转为切片
	var specSummaries []models.SpecSummaryResponse
	for _, summary := range specSummaryMap {
		specSummaries = append(specSummaries, *summary)
	}

	// ---------- 2. 获取用户默认地址与运费 ----------
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

	// 计算运费（基于商品总额）
	freight := utils.CalculateFreight(totalAmount)
	totalWithFreight := totalAmount + freight

	// ---------- 3. 优惠券列表拉取与计算 ----------
	var userCoupons []models.UserCoupon
	now := time.Now()
	// 查询用户所有未使用且未过期的优惠券（预加载模板信息）
	database.DB.Where("user_id = ? AND status = ? AND valid_start <= ? AND valid_end >= ?",
		userID, models.UserCouponUnused, now, now).
		Preload("Coupon").
		Find(&userCoupons)

	type calcCoupon struct {
		UserCoupon models.UserCoupon
		Discount   float64 // 该券在当前订单可优惠的金额
		Status     int     // 1-可用 0-不可用
		Reason     string  // 不可用原因
	}
	var tempCoupons []calcCoupon
	var bestUserCouponID string
	var bestDiscount float64

	// 逐张券校验可用性并计算优惠金额
	for _, uc := range userCoupons {
		disc, err := services.ValidateUserCoupon(uc, totalWithFreight, []string{req.ProductID})
		if err != nil {
			tempCoupons = append(tempCoupons, calcCoupon{
				UserCoupon: uc,
				Discount:   0,
				Status:     0,
				Reason:     err.Error(),
			})
			continue
		}
		tempCoupons = append(tempCoupons, calcCoupon{
			UserCoupon: uc,
			Discount:   disc,
			Status:     1,
			Reason:     "",
		})
		if disc > bestDiscount {
			bestDiscount = disc
			bestUserCouponID = uc.ID.String()
		}
	}

	// 确定最终选中的用户券实例ID
	selectedUserCouponID := ""
	if req.CouponID != "" {
		// 1. 优先匹配用户券实例ID
		for _, cp := range tempCoupons {
			if cp.UserCoupon.ID.String() == req.CouponID && cp.Status == 1 {
				selectedUserCouponID = req.CouponID
				break
			}
		}
		// 2. 若未匹配到，尝试作为模板ID匹配（从同模板的可用券中选优惠金额最大的）
		if selectedUserCouponID == "" {
			var bestInTemplate *calcCoupon
			for i, cp := range tempCoupons {
				if cp.UserCoupon.CouponID.String() == req.CouponID && cp.Status == 1 {
					if bestInTemplate == nil || cp.Discount > bestInTemplate.Discount {
						bestInTemplate = &tempCoupons[i]
					}
				}
			}
			if bestInTemplate != nil {
				selectedUserCouponID = bestInTemplate.UserCoupon.ID.String()
			}
		}
	}
	// 若用户未传券或传的券无效，则使用系统自动选择的最佳券
	if selectedUserCouponID == "" && bestUserCouponID != "" {
		selectedUserCouponID = bestUserCouponID
	}

	// 构建返回的优惠券列表（包含可用/不可用券）
	var couponList []PreviewCouponResponse
	for _, cp := range tempCoupons {
		uc := cp.UserCoupon
		tpl := uc.Coupon
		amountDesc := ""
		switch tpl.Type {
		case models.CouponTypeFullReduce:
			amountDesc = fmt.Sprintf("满%.0f减%.0f", tpl.FullAmount, tpl.ReduceAmount)
		case models.CouponTypeNoThreshold:
			amountDesc = fmt.Sprintf("减%.0f", tpl.ReduceAmount)
		case models.CouponTypeDiscount:
			amountDesc = fmt.Sprintf("%.1f折", tpl.DiscountRate*10)
		}
		couponList = append(couponList, PreviewCouponResponse{
			UserCouponID: uc.ID.String(),
			ID:           tpl.ID.String(),
			Name:         tpl.Name,
			FullAmount:   tpl.FullAmount,
			ReduceAmount: tpl.ReduceAmount,
			DiscountRate: tpl.DiscountRate,
			MaxReduce:    tpl.MaxReduce,
			Type:         int(tpl.Type),
			AmountDesc:   amountDesc,
			MinAmount:    tpl.FullAmount,
			ValidStart:   uc.ValidStart.Format("2006-01-02 15:04:05"),
			ValidEnd:     uc.ValidEnd.Format("2006-01-02 15:04:05"),
			Status:       cp.Status,
			Reason:       cp.Reason,
			Discount:     cp.Discount,
		})
	}

	// 排序：选中的券排在第一位，其余按优惠金额降序
	sort.Slice(couponList, func(i, j int) bool {
		if couponList[i].UserCouponID == selectedUserCouponID {
			return true
		}
		if couponList[j].UserCouponID == selectedUserCouponID {
			return false
		}
		return couponList[i].Discount > couponList[j].Discount
	})

	// 计算最终选中的优惠券的减免金额
	finalDiscount := 0.0
	for _, cp := range couponList {
		if cp.UserCouponID == selectedUserCouponID {
			finalDiscount = cp.Discount
			break
		}
	}

	finalAmount := totalWithFreight - finalDiscount
	if finalAmount < 0 {
		finalAmount = 0
	}

	// ---------- 4. 返回结果 ----------
	utils.Success(c, PreviewOrderResponseData{
		Items:            previewItems,         // 商品预览列表，每个元素包含商品ID、名称、规格、价格、数量、小计、图片URL
		Specs:            specSummaries,        // 规格汇总数组，每个元素包含商品ID、商品名称、规格ID、规格名称、单价、总数量、总小计、图片URL
		Coupons:          couponList,           // 用户可用优惠券列表，每个元素包含用户券ID、模板ID、名称、满减门槛、减额、折扣率、类型、优惠文案、最低消费、有效期、状态、不可用原因、可优惠金额
		TotalAmount:      totalAmount,          // 商品总金额（不含运费）
		Freight:          freight,              // 运费金额
		DiscountAmount:   finalDiscount,        // 最终使用的优惠券减免金额
		ActualAmount:     finalAmount,          // 实付金额 = 商品总金额 + 运费 - 优惠金额
		DefaultAddress:   addressResp,          // 用户默认收货地址，包含地址ID、收件人、手机号、省市区名称、详细地址、门牌号
		SelectedCouponId: selectedUserCouponID, // 最终选中的用户优惠券实例ID，若未使用任何券则为空字符串
	})
}
