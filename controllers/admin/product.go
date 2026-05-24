package admin

import (
	"encoding/json"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 创建商品请求结构
type CreateProductReq struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description"`
	Detail       string                 `json:"detail"`
	CoverImage   string                 `json:"coverImage"`   // 封面图
	BannerImages []string               `json:"bannerImages"` // 轮播图数组
	Status       models.ProductStatus   `json:"status"`
	SortOrder    int                    `json:"sortOrder"`
	Specs        []CreateProductSpecReq `json:"specs" binding:"required,min=1"`
}

type CreateProductSpecReq struct {
	Name      string  `json:"name" binding:"required"`
	Image     string  `json:"image"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Stock     int     `json:"stock"`
	SkuCode   string  `json:"skuCode"`
	SortOrder int     `json:"sortOrder"`
}

// CreateProduct 新建商品
// @Summary 新建商品
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param product body CreateProductReq true "商品信息"
// @Success 200 {object} utils.Response{data=models.Product}
// @Router /api/v1/admin/products [post]
func CreateProduct(c *gin.Context) {
	var req CreateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	product := models.Product{
		Name:         req.Name,
		Description:  req.Description,
		Detail:       req.Detail,
		CoverImage:   req.CoverImage,
		BannerImages: req.BannerImages,
		Status:       req.Status,
		SortOrder:    req.SortOrder,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&product).Error; err != nil {
			return err
		}
		for _, s := range req.Specs {
			spec := models.ProductSpec{
				ProductID: product.ID,
				Name:      s.Name,
				Image:     s.Image,
				Price:     s.Price,
				Stock:     s.Stock,
				SkuCode:   s.SkuCode,
				SortOrder: s.SortOrder,
			}
			if err := tx.Create(&spec).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "创建商品失败")
		return
	}

	// 预加载规格后返回
	database.DB.Preload("Specs").First(&product, product.ID)
	// utils.Success(c, product)
	utils.Success(c, nil)
}

// UpdateProductReq 更新商品请求
type UpdateProductReq struct {
	Name         *string                `json:"name"`
	Description  *string                `json:"description"`
	Detail       *string                `json:"detail"`
	CoverImage   *string                `json:"coverImage"`   // 指针允许未传时不更新
	BannerImages *[]string              `json:"bannerImages"` // 指针
	Status       *models.ProductStatus  `json:"status"`
	SortOrder    *int                   `json:"sortOrder"`
	Specs        []UpdateProductSpecReq `json:"specs"`
}

type UpdateProductSpecReq struct {
	ID        *utils.Int64Str `json:"id"` // 更新时传入ID，新增时为空
	Name      string          `json:"name" binding:"required"`
	Image     string          `json:"image"`
	Price     float64         `json:"price" binding:"required,gt=0"`
	Stock     int             `json:"stock"`
	SkuCode   string          `json:"skuCode"`
	SortOrder int             `json:"sortOrder"`
}

// UpdateProduct 修改商品
// @Summary 修改商品
// @Tags 商品管理
// @Param id path int true "商品ID"
// @Param product body UpdateProductReq true "商品信息"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/products/{id} [put]
func UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		utils.Fail(c, "商品不存在")
		return
	}

	var req UpdateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	// 更新商品基本信息
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Detail != nil {
		updates["detail"] = *req.Detail
	}
	if req.CoverImage != nil {
		updates["cover_image"] = *req.CoverImage
	}
	if req.BannerImages != nil {
		jsonData, _ := json.Marshal(*req.BannerImages) // 省略 err 处理需补全
		updates["banner_images"] = string(jsonData)    // 关键修复
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if len(updates) > 0 {
		if err := database.DB.Model(&product).Updates(updates).Error; err != nil {
			utils.Fail(c, "更新商品失败")
			return
		}
	}

	// 处理规格：全量替换（简化逻辑，也可实现部分更新）
	if req.Specs != nil {
		// 删除原有规格
		if err := database.DB.Where("product_id = ?", product.ID).Delete(&models.ProductSpec{}).Error; err != nil {
			utils.Fail(c, "更新规格失败")
			return
		}
		// 新建规格
		for _, s := range req.Specs {
			spec := models.ProductSpec{
				ProductID: product.ID,
				Name:      s.Name,
				Image:     s.Image,
				Price:     s.Price,
				Stock:     s.Stock,
				SkuCode:   s.SkuCode,
				SortOrder: s.SortOrder,
			}
			if err := database.DB.Create(&spec).Error; err != nil {
				utils.Fail(c, "更新规格失败")
				return
			}
		}
	}

	utils.Success(c, nil)
}

// GetProductList 商品列表（后台）
// @Summary 商品列表
// @Tags 商品管理
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param status query string false "状态"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} utils.Response{data=object{list=[]models.Product,total=int64,page=int,size=int}}
// @Router /api/v1/admin/products [get]
func GetProductList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	status := c.Query("status")
	name := c.Query("name")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size

	var products []models.Product
	var total int64

	query := database.DB.Model(&models.Product{}).Preload("Specs")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("sort_order asc, created_at desc").Find(&products)

	utils.Success(c, gin.H{
		"list":  products,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetProductDetail 商品详情（后台/小程序通用）
// @Summary 商品详情
// @Tags 商品管理
// @Param id path int true "商品ID"
// @Success 200 {object} utils.Response{data=models.Product}
// @Router /api/v1/admin/products/{id} [get]
func GetProductDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var product models.Product
	if err := database.DB.Preload("Specs").First(&product, id).Error; err != nil {
		utils.Fail(c, "商品不存在")
		return
	}
	utils.Success(c, product)
}

// UpdateProductStatus 修改商品状态（上架/下架）
// @Summary 修改商品状态
// @Tags 商品管理
// @Param id path int true "商品ID"
// @Param status body object true "状态"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/products/{id}/status [put]
func UpdateProductStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var req struct {
		Status models.ProductStatus `json:"status" binding:"required,oneof=draft on_sale off_sale"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "状态值无效")
		return
	}
	result := database.DB.Model(&models.Product{}).Where("id = ?", id).Update("status", req.Status)
	if result.RowsAffected == 0 {
		utils.Fail(c, "商品不存在")
		return
	}
	utils.Success(c, nil)
}

// DeleteProduct 删除商品（软删除或物理删除，推荐软删除，这里简单物理删除）
// @Summary 删除商品
// @Tags 商品管理
// @Param id path int true "商品ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/products/{id} [delete]
func DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	// 删除商品时一并删除关联规格（事务）
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", id).Delete(&models.ProductSpec{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Product{}, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		utils.Fail(c, "删除失败")
		return
	}
	utils.Success(c, nil)
}
