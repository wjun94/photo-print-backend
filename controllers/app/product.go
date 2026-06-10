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

// GetProductDetailForWx 小程序商品详情
// @Summary      小程序商品详情
// @Description  根据商品ID获取详情，包含所有规格、轮播图、描述、跳转动作（confirm/upload）等
// @Tags         小程序-商品
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "商品ID"
// @Success      200  {object}  utils.Response{data=models.Product}
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
	product.FreeShippingAmount = config.AppConfig.FreeShippingAmount
	// 直接返回完整商品对象，其中包含 Action 字段
	utils.Success(c, product)
}
