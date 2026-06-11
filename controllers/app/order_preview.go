package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/services"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PreviewOrderItem struct {
	ImageURL string `json:"imageUrl"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type PreviewOrderReq struct {
	ProductID string             `json:"productId" binding:"required"`
	SpecID    string             `json:"specId" binding:"required"`
	Items     []PreviewOrderItem `json:"items" binding:"required,min=1"`
	CouponID  string             `json:"couponId"` // 传入代表用户手动切换/指定券
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

type PreviewCouponResponse struct {
	UserCouponID string  `json:"userCouponId"`
	CouponID     string  `json:"couponId"`
	Name         string  `json:"name"`
	Type         int     `json:"type"`
	AmountDesc   string  `json:"amountDesc"`
	MinAmount    float64 `json:"minAmount"`
	ValidStart   string  `json:"validStart"`
	ValidEnd     string  `json:"validEnd"`
	Selectable   bool    `json:"selectable"` // 是否满足当前订单使用条件
	Reason       string  `json:"reason"`     // 不可用原因
}

// PreviewOrder 确认订单页面预览（支持返回券列表及自动选最佳券）
// @Summary 订单预览
// @Description 根据商品、规格、数量计算金额，返回可用/不可用券列表。若提供couponId则使用指定券，否则自动选择最佳可用券
// @Tags 订单
// @Accept json
// @Produce json
// @Param request body PreviewOrderReq true "预览请求"
// @Success 200 {object} utils.Response{data=object{items=[]PreviewItemResponse,specs=[]models.SpecSummaryResponse,coupons=[]PreviewCouponResponse,totalAmount=float64,freight=float64,discountAmount=float64,actualAmount=float64,defaultAddress=object,selectedCouponId=string}}
// @Router /api/v1/wx/order/preview [post]
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

	// ---------- 1. 商品校验与金额计算 ----------
	var previewItems []PreviewItemResponse
	var totalAmount float64
	specSummaryMap := make(map[string]*models.SpecSummaryResponse)

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

	freight := utils.CalculateFreight(totalAmount)
	totalWithFreight := totalAmount + freight // 注意底层计算优惠券门槛一般不包含运费，可根据业务传入 totalAmount 或 totalWithFreight

	// ---------- 3. 优惠券列表拉取与多维度校验 ----------
	var userCoupons []models.UserCoupon
	now := time.Now()
	// 拉取该用户名下所有：未使用、在有效期内 的优惠券记录
	database.DB.Where("user_id = ? AND status = ? AND valid_start <= ? AND valid_end >= ?",
		userID, models.UserCouponUnused, now, now).
		Preload("Coupon").
		Find(&userCoupons)

	var couponListResp []PreviewCouponResponse
	var bestUserCouponID string
	var maxDiscountAmount float64

	// 遍历用户持有的优惠券，动态判断在当前商品及金额下是否满足条件
	for _, uc := range userCoupons {
		selectable := true
		reason := ""
		discount := 0.0

		// 调用底层服务进行校验（传入当前购买的 productID 列表和总额）
		// 提示：此处传入 totalAmount (商品总价) 还是 totalWithFreight 依公司业务规则而定
		disc, err := services.ValidateUserCoupon(uc, totalAmount, []string{req.ProductID})
		if err != nil {
			selectable = false
			reason = err.Error()
		} else {
			discount = disc
		}

		// 格式化金额说明描述
		amountDesc := ""
		switch uc.Coupon.Type {
		case models.CouponTypeFullReduce:
			amountDesc = fmt.Sprintf("减%.2f元", uc.Coupon.ReduceAmount)
		case models.CouponTypeNoThreshold:
			amountDesc = fmt.Sprintf("无门槛减%.2f元", uc.Coupon.ReduceAmount)
		case models.CouponTypeDiscount:
			amountDesc = fmt.Sprintf("%.1f折", uc.Coupon.DiscountRate*10)
		}

		couponListResp = append(couponListResp, PreviewCouponResponse{
			UserCouponID: uc.ID.String(),
			CouponID:     uc.CouponID.String(),
			Name:         uc.Coupon.Name,
			Type:         int(uc.Coupon.Type),
			AmountDesc:   amountDesc,
			MinAmount:    uc.Coupon.FullAmount,
			ValidStart:   uc.ValidStart.Format("2006-01-02 15:04:05"),
			ValidEnd:     uc.ValidEnd.Format("2006-01-02 15:04:05"),
			Selectable:   selectable,
			Reason:       reason,
		})

		// 核心策略：如果用户没传 couponId，在遍历过程中动态选出【满足条件且减免金额最大】的最佳券
		if req.CouponID == "" && selectable && discount > maxDiscountAmount {
			maxDiscountAmount = discount
			bestUserCouponID = uc.CouponID.String() // 或者是 uc.ID.String()，取决于前端下单时传模板ID还是用户持有券实例ID
		}
	}

	// ---------- 4. 最终选中的优惠券优惠金额计算 ----------
	discountAmount := 0.0
	selectedCouponID := ""

	if req.CouponID != "" {
		// 情况 A：用户在前端手动挑选/指定了某张券
		selectedCouponID = req.CouponID
		cid, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err != nil {
			utils.Fail(c, "无效优惠券ID")
			return
		}

		var targetUC models.UserCoupon
		if err := database.DB.Where("user_id = ? AND coupon_id = ? AND status = ?", userID, cid, models.UserCouponUnused).
			Preload("Coupon").First(&targetUC).Error; err != nil {
			utils.Fail(c, "指定的优惠券不存在或已不可用")
			return
		}

		disc, err := services.ValidateUserCoupon(targetUC, totalAmount, []string{req.ProductID})
		if err != nil {
			utils.Fail(c, "不可使用该券: "+err.Error())
			return
		}
		discountAmount = disc
	} else {
		// 情况 B：用户未选券，使用刚才遍历计算出的最佳优惠券
		if bestUserCouponID != "" {
			selectedCouponID = bestUserCouponID
			discountAmount = maxDiscountAmount
		}
	}

	// ---------- 5. 金额汇总并输出 ----------
	finalAmount := totalWithFreight - discountAmount
	if finalAmount < 0 {
		finalAmount = 0
	}

	utils.Success(c, gin.H{
		"items":            previewItems,     // 商品明细列表
		"specs":            specSummaries,    // 规格汇总（按商品规格聚合）
		"coupons":          couponListResp,   // 新增：返回所有的已领取优惠券列表及可用状态判断
		"totalAmount":      totalAmount,      // 商品总金额（未含运费，未减优惠）
		"freight":          freight,          // 运费金额
		"discountAmount":   discountAmount,   // 优惠券抵扣金额
		"actualAmount":     finalAmount,      // 实际支付金额（总金额+运费-优惠）
		"defaultAddress":   addressResp,      // 用户默认地址对象（无地址时为null）
		"selectedCouponId": selectedCouponID, // 当前选中的优惠券实例ID（自动最佳或用户指定）
	})
}
