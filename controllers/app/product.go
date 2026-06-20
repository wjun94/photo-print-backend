package app

import (
	"fmt"
	"photo-print-backend/config"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProductListItem 小程序商品列表项（精简字段）
type ProductListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CoverImage  string `json:"coverImage"`
	Price       string `json:"price"`       // 格式化后的最低价文本，如 "19.99"
	PriceSuffix string `json:"priceSuffix"` // 价格后缀，如 "起" 或空
}

// GetProductListForWx 小程序商品列表（仅上架商品）
// @Summary      小程序商品列表
// @Description  获取已上架的商品列表，按排序序号正序、创建时间倒序排列，返回最低价及跳转动作
// @Tags         小程序-商品
// @Accept       json
// @Produce      json
// @Param        page  query   int     false  "页码，默认1"
// @Param        size  query   int     false  "每页数量，默认10，最大20"
// @Success      200   {object} utils.Response{data=object{list=[]ProductListItem,total=int64,page=int,size=int}}
// @Router       /api/v1/wx/products [get]
func GetProductListForWx(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 20 {
		size = 10
	}
	offset := (page - 1) * size

	var products []models.Product
	var total int64

	// 仅查询上架商品，并预加载规格
	query := database.DB.Model(&models.Product{}).
		Where("status = ?", models.ProductStatusOnSale).
		Preload("Specs")
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("sort_order asc, created_at desc").Find(&products)

	// 构建返回列表
	list := make([]ProductListItem, 0, len(products))
	for _, p := range products {
		if len(p.Specs) == 0 {
			list = append(list, ProductListItem{
				ID:          p.ID.String(),
				Name:        p.Name,
				CoverImage:  p.CoverImage,
				Price:       "0.00",
				PriceSuffix: "",
			})
			continue
		}

		// 计算最低价及价格集合
		minPrice := p.Specs[0].Price
		priceSet := make(map[float64]struct{})
		priceSet[minPrice] = struct{}{}
		for _, spec := range p.Specs[1:] {
			if spec.Price < minPrice {
				minPrice = spec.Price
			}
			priceSet[spec.Price] = struct{}{}
		}

		var price string
		var priceSuffix string
		if len(priceSet) == 1 {
			price = fmt.Sprintf("%.2f", minPrice)
			priceSuffix = ""
		} else {
			price = fmt.Sprintf("%.2f", minPrice)
			priceSuffix = "起"
		}

		list = append(list, ProductListItem{
			ID:          p.ID.String(),
			Name:        p.Name,
			CoverImage:  p.CoverImage,
			Price:       price,
			PriceSuffix: priceSuffix,
		})
	}

	utils.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// ProductDetailResponse 商品详情响应（平铺结构）
type ProductDetailResponse struct {
	ID           utils.Int64Str       `gorm:"primarykey;autoIncrement:false" json:"id"` // 商品 ID，雪花算法生成
	Name         string               `gorm:"size:200;not null" json:"name"`            // 商品名称
	CoverImage   string               `gorm:"size:500" json:"coverImage"`               // 封面图片 URL
	BannerImages models.StringArray   `gorm:"type:json" json:"bannerImages"`            // 轮播图 URL 数组，JSON 存储
	Description  string               `gorm:"type:text" json:"description"`             // 简短描述，用于列表页
	Detail       string               `gorm:"type:longtext" json:"detail"`              // 富文本详情（HTML/Markdown）
	Status       models.ProductStatus `gorm:"default:'draft';size:20" json:"status"`    // 商品状态：draft, on_sale, off_sale
	SortOrder    int                  `gorm:"default:0" json:"sortOrder"`               // 排序序号，数字越小越靠前
	CreatedAt    utils.LocalTime      `json:"createdAt"`                                // 创建时间
	UpdatedAt    utils.LocalTime      `json:"updatedAt"`                                // 更新时间
	Action       models.ProductAction `gorm:"size:20;default:'confirm'" json:"action"`  // confirm-确认订单, upload-上传照片
	Tags         models.StringArray   `gorm:"type:json;serializer:json" json:"tags"`    // 商品标签，JSON 数组存储，如 ["热销","新品"]

	// 关联的规格属性模板（如颜色、尺寸的可选值）
	SpecAttributes []models.SpecAttribute `gorm:"foreignKey:ProductID" json:"specAttributes,omitempty"`
	// 具体的 SKU 规格列表（每个规格包含价格、库存及属性组合）
	Specs              []models.Spec `gorm:"foreignKey:ProductID;references:ID" json:"specs,omitempty"`
	FreeShippingAmount float64       `gorm:"-" json:"freeShippingAmount,omitempty"` // 包邮门槛（临时字段，不存数据库）

	MinPrice float64 `json:"minPrice"` // 最低价格
	MaxPrice float64 `json:"maxPrice"` // 最高价格
}

// GetProductDetailForWx 小程序商品详情
// @Summary      小程序商品详情
// @Description  根据商品ID获取详情，包含所有规格、轮播图、描述、跳转动作（confirm/upload）等
// @Tags         小程序-商品
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "商品ID"
// @Success      200  {object}  utils.Response{data=ProductDetailResponse}
// @Failure      400  {object}  utils.Response
// @Failure      404  {object}  utils.Response
// @Router       /api/v1/wx/products/{id} [get]
func GetProductDetailForWx(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var product models.Product
	if err := database.DB.
		Where("status = ?", models.ProductStatusOnSale).
		Preload("Specs").
		Preload("SpecAttributes").
		First(&product, id).Error; err != nil {
		utils.Fail(c, "商品不存在或已下架")
		return
	}

	// 计算价格区间
	minPrice := 0.0
	maxPrice := 0.0
	if len(product.Specs) > 0 {
		minPrice = product.Specs[0].Price
		// maxPrice = product.Specs[0].Price
		for _, spec := range product.Specs[1:] {
			if spec.Price < minPrice {
				minPrice = spec.Price
			}
			if spec.Price > maxPrice && spec.Price > minPrice {
				maxPrice = spec.Price
			}
		}
	}
	product.FreeShippingAmount = config.AppConfig.FreeShippingAmount

	// 构建响应（直接平铺，不包裹 product）
	resp := ProductDetailResponse{
		ID:                 product.ID,
		Name:               product.Name,
		CoverImage:         product.CoverImage,
		BannerImages:       product.BannerImages,
		Description:        product.Description,
		Detail:             product.Detail,
		Status:             product.Status,
		SortOrder:          product.SortOrder,
		CreatedAt:          product.CreatedAt,
		UpdatedAt:          product.UpdatedAt,
		Action:             product.Action,
		Tags:               product.Tags,
		SpecAttributes:     product.SpecAttributes,
		Specs:              product.Specs,
		FreeShippingAmount: config.AppConfig.FreeShippingAmount,
		MinPrice:           minPrice,
		MaxPrice:           maxPrice,
	}

	// 直接返回完整商品对象，其中包含 Action 字段
	utils.Success(c, resp)
}
