package app

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CoverImage  string `json:"coverImage"`
	Price       string `json:"price"` // 格式化后的价格文本
	PriceSuffix string `json:"priceSuffix"`
}

// GetProductListForWx 小程序商品列表（仅上架商品，精简字段）
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
			// 无规格的商品，价格显示 0 或自定义
			list = append(list, ProductListItem{
				ID:          p.ID.String(),
				Name:        p.Name,
				CoverImage:  p.CoverImage,
				Price:       "0",
				PriceSuffix: "",
			})
			continue
		}

		// 计算最低价和价格集合
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
			// 所有规格价格相同
			price = fmt.Sprintf("%.2f", minPrice)
			priceSuffix = ""
		} else {
			// 存在不同价格
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
func GetProductDetailForWx(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var product models.Product
	if err := database.DB.Where("status = ?", models.ProductStatusOnSale).Preload("Specs").First(&product, id).Error; err != nil {
		utils.Fail(c, "商品不存在或已下架")
		return
	}
	utils.Success(c, product)
}
