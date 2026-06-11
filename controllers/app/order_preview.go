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
	CouponID  string             `json:"couponId"` // 用户优惠券实例ID，可选
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
	Status       int     `json:"status"` // 1-可使用，0-不可使用
	Reason       string  `json:"reason"` // 不可使用原因
}

// PreviewOrder 确认订单页面预览
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
	totalWithFreight := totalAmount + freight

	// ---------- 3. 优惠券列表拉取与多维度校验 ----------
	var userCoupons []models.UserCoupon
	now := time.Now()
	database.DB.Where("user_id = ? AND status = ? AND valid_start <= ? AND valid_end >= ?",
		userID, models.UserCouponUnused, now, now).
		Preload("Coupon").
		Find(&userCoupons)

	var couponListResp []PreviewCouponResponse
	var bestUserCouponID string
	var maxDiscountAmount float64

	for _, uc := range userCoupons {
		selectable := true
		reason := ""
		discount := 0.0

		disc, err := services.ValidateUserCoupon(uc, totalAmount, []string{req.ProductID})
		if err != nil {
			selectable = false
			reason = err.Error()
		} else {
			discount = disc
		}

		amountDesc := ""
		switch uc.Coupon.Type {
		case models.CouponTypeFullReduce:
			amountDesc = fmt.Sprintf("减%.2f元", uc.Coupon.ReduceAmount)
		case models.CouponTypeNoThreshold:
			amountDesc = fmt.Sprintf("无门槛减%.2f元", uc.Coupon.ReduceAmount)
		case models.CouponTypeDiscount:
			amountDesc = fmt.Sprintf("%.1f折", uc.Coupon.DiscountRate*10)
		}

		status := 0
		if selectable {
			status = 1
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
			Status:       status,
			Reason:       reason,
		})

		// 自动选择最佳可用券（selectable且抵扣金额最大）
		if req.CouponID == "" && selectable && discount > maxDiscountAmount {
			maxDiscountAmount = discount
			bestUserCouponID = uc.ID.String() // 存储用户券实例ID
		}
	}

	// ---------- 4. 最终选中的优惠券优惠金额计算 ----------
	discountAmount := 0.0
	selectedCouponID := ""

	if req.CouponID != "" {
		selectedCouponID = req.CouponID
		userCouponID, err := strconv.ParseInt(req.CouponID, 10, 64)
		if err != nil {
			utils.Fail(c, "无效优惠券ID")
			return
		}
		var targetUC models.UserCoupon
		if err := database.DB.Where("coupon_id = ? AND user_id = ? AND status = ?", userCouponID, userID, models.UserCouponUnused).
			Preload("Coupon").First(&targetUC).Error; err != nil {
			utils.Fail(c, "指定的优惠券不存在或已不可用")
			return
		}
		disc, err := services.ValidateUserCoupon(targetUC, totalAmount, []string{req.ProductID})
		if err != nil {
			// utils.Fail(c, "不可使用该券: "+err.Error())
			// return
			discountAmount = disc
		} else {
			discountAmount = 0
		}
	} else {
		if bestUserCouponID != "" {
			selectedCouponID = bestUserCouponID
			discountAmount = maxDiscountAmount
		}
	}

	// ---------- 5. 金额汇总 ----------
	finalAmount := totalWithFreight - discountAmount
	if finalAmount < 0 {
		finalAmount = 0
	}

	utils.Success(c, gin.H{
		"items":            previewItems,
		"specs":            specSummaries,
		"coupons":          couponListResp,
		"totalAmount":      totalAmount,
		"freight":          freight,
		"discountAmount":   discountAmount,
		"actualAmount":     finalAmount,
		"defaultAddress":   addressResp,
		"selectedCouponId": selectedCouponID,
	})
}
