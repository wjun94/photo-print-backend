package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetProductListForWx 小程序商品列表（仅上架商品）
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
	query := database.DB.Model(&models.Product{}).Where("status = ?", models.ProductStatusOnSale).Preload("Specs")
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("sort_order asc, created_at desc").Find(&products)

	utils.Success(c, gin.H{
		"list":  products,
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
