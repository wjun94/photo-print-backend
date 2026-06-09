package admin

import (
	"encoding/json"
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SpecAttributeReq 规格属性模板请求
type SpecAttributeReq struct {
	Name   string   `json:"name" binding:"required"`   // 属性名，如 "颜色"
	Values []string `json:"values" binding:"required"` // 可选值列表，如 ["标准","绿色"]
}

// SpecReq SKU 规格请求
type SpecReq struct {
	SkuKey     string            `json:"skuKey"` // 规格组合唯一键，如 "颜色:标准_尺寸:5寸"
	Price      float64           `json:"price" binding:"required,gt=0"`
	Stock      int               `json:"stock" binding:"min=0"`
	SkuCode    string            `json:"skuCode"`
	Image      string            `json:"image"`
	Attributes map[string]string `json:"attributes"` // 动态属性，如 {"颜色":"标准","尺寸":"5寸"}
}

// UnmarshalJSON 自定义反序列化，将未匹配的字段放入 Attributes
func (s *SpecReq) UnmarshalJSON(data []byte) error {
	type Alias SpecReq
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	// 先解析已知字段
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// 再解析所有字段，提取未知字段
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if s.Attributes == nil {
		s.Attributes = make(map[string]string)
	}
	for k, v := range raw {
		// 跳过已知字段
		if k == "id" || k == "skuKey" || k == "price" || k == "stock" || k == "skuCode" || k == "image" || k == "attributes" {
			continue
		}
		// 将值转为字符串
		if strVal, ok := v.(string); ok {
			s.Attributes[k] = strVal
		} else {
			s.Attributes[k] = fmt.Sprintf("%v", v)
		}
	}
	return nil
}

// CreateProductReq 创建商品请求
type CreateProductReq struct {
	Name           string               `json:"name" binding:"required"`
	CoverImage     string               `json:"coverImage"`
	BannerImages   []string             `json:"bannerImages"`
	Description    string               `json:"description"`
	Detail         string               `json:"detail"`
	Status         models.ProductStatus `json:"status" binding:"oneof=draft on_sale off_sale"`
	SortOrder      int                  `json:"sortOrder"`
	SpecAttributes []SpecAttributeReq   `json:"specAttributes"`
	Specs          []SpecReq            `json:"specs" binding:"required,min=1"`
}

// CreateProduct 创建商品（支持多规格）
// @Summary 创建商品
// @Description 创建商品，包含规格属性模板和 SKU 列表
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

	// 商品主表数据
	product := models.Product{
		Name:         req.Name,
		CoverImage:   req.CoverImage,
		BannerImages: models.StringArray(req.BannerImages),
		Description:  req.Description,
		Detail:       req.Detail,
		Status:       req.Status,
		SortOrder:    req.SortOrder,
		CreatedAt:    utils.LocalTime(time.Now()),
		UpdatedAt:    utils.LocalTime(time.Now()),
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 创建商品
		if err := tx.Create(&product).Error; err != nil {
			return err
		}

		// 2. 创建规格属性模板
		for i, attr := range req.SpecAttributes {
			specAttr := models.SpecAttribute{
				ProductID: product.ID,
				Name:      attr.Name,
				Values:    models.StringArray(attr.Values),
				SortOrder: i,
			}
			if err := tx.Create(&specAttr).Error; err != nil {
				return err
			}
		}

		// 3. 创建 SKU 规格
		for _, s := range req.Specs {
			// 自动生成 skuKey（如果未提供）
			skuKey := s.SkuKey
			if skuKey == "" && len(s.Attributes) > 0 {
				parts := make([]string, 0, len(s.Attributes))
				for k, v := range s.Attributes {
					parts = append(parts, fmt.Sprintf("%s:%s", k, v))
				}
				skuKey = strings.Join(parts, ",")
			}
			spec := models.Spec{
				ProductID:  product.ID,
				SkuKey:     skuKey,
				Price:      s.Price,
				Stock:      s.Stock,
				SkuCode:    s.SkuCode,
				Image:      s.Image,
				Attributes: models.StringMap(s.Attributes),
			}
			if err := tx.Create(&spec).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		utils.Fail(c, "创建商品失败: "+err.Error())
		return
	}

	// 预加载关联数据后返回
	database.DB.Preload("SpecAttributes").Preload("Specs").First(&product, product.ID)
	utils.Success(c, product)
}

// UpdateProductReq 更新商品请求（支持部分更新）
type UpdateProductReq struct {
	Name           *string               `json:"name"`
	CoverImage     *string               `json:"coverImage"`
	BannerImages   *[]string             `json:"bannerImages"`
	Description    *string               `json:"description"`
	Detail         *string               `json:"detail"`
	Status         *models.ProductStatus `json:"status" binding:"omitempty,oneof=draft on_sale off_sale"`
	SortOrder      *int                  `json:"sortOrder"`
	SpecAttributes *[]SpecAttributeReq   `json:"specAttributes"` // 全量替换规格属性
	Specs          *[]SpecReq            `json:"specs"`          // 全量替换 SKU
}

// UpdateProduct 更新商品（支持全量替换规格属性和 SKU）
// @Summary 更新商品
// @Description 更新商品信息，规格属性和 SKU 会全量替换
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param id path int true "商品ID"
// @Param product body UpdateProductReq true "商品信息"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/products/{id} [put]
func UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效商品ID")
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

	// 更新商品主表字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.CoverImage != nil {
		updates["cover_image"] = *req.CoverImage
	}
	if req.BannerImages != nil {
		updates["banner_images"] = models.StringArray(*req.BannerImages)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Detail != nil {
		updates["detail"] = *req.Detail
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if len(updates) > 0 {
		updates["updated_at"] = utils.LocalTime(time.Now())
		if err := database.DB.Model(&product).Updates(updates).Error; err != nil {
			utils.Fail(c, "更新商品失败")
			return
		}
	}

	// 处理规格属性和 SKU 的全量替换（事务）
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 如果提供了规格属性，则删除旧的并重新创建
		if req.SpecAttributes != nil {
			if err := tx.Where("product_id = ?", product.ID).Delete(&models.SpecAttribute{}).Error; err != nil {
				return err
			}
			for i, attr := range *req.SpecAttributes {
				specAttr := models.SpecAttribute{
					ProductID: product.ID,
					Name:      attr.Name,
					Values:    models.StringArray(attr.Values),
					SortOrder: i,
				}
				if err := tx.Create(&specAttr).Error; err != nil {
					return err
				}
			}
		}

		// 如果提供了 SKU 列表，则删除旧的并重新创建
		if req.Specs != nil {
			if err := tx.Where("product_id = ?", product.ID).Delete(&models.Spec{}).Error; err != nil {
				return err
			}
			for _, s := range *req.Specs {
				skuKey := s.SkuKey
				if skuKey == "" && len(s.Attributes) > 0 {
					parts := make([]string, 0, len(s.Attributes))
					for k, v := range s.Attributes {
						parts = append(parts, fmt.Sprintf("%s:%s", k, v))
					}
					skuKey = strings.Join(parts, ",")
				}
				spec := models.Spec{
					ProductID:  product.ID,
					SkuKey:     skuKey,
					Price:      s.Price,
					Stock:      s.Stock,
					SkuCode:    s.SkuCode,
					Image:      s.Image,
					Attributes: models.StringMap(s.Attributes),
				}
				if err := tx.Create(&spec).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		utils.Fail(c, "更新规格失败: "+err.Error())
		return
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

// GetProductDetail 获取商品详情（后台）
// @Summary 获取商品详情
// @Description 根据商品ID返回详细信息，包括规格属性模板和SKU列表
// @Tags 商品管理
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} utils.Response{data=models.Product}
// @Router /api/v1/admin/products/{id} [get]
func GetProductDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效商品ID")
		return
	}

	var product models.Product
	// 预加载关联的规格属性模板和SKU规格
	err = database.DB.
		Preload("SpecAttributes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("Specs").
		First(&product, id).Error
	if err != nil {
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

// DeleteProductReq 删除商品请求参数
type DeleteProductReq struct {
	ProductID string `json:"productId" binding:"required"`
}

// DeleteProduct 删除商品（管理员）
// @Summary 删除商品
// @Description 物理删除商品及其关联的规格属性、SKU。若已被订单引用则禁止删除。
// @Tags 商品管理
// @Accept json
// @Produce json
// @Param request body DeleteProductReq true "商品ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Router /api/v1/admin/product/delete [post]
func DeleteProduct(c *gin.Context) {
	var req DeleteProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}

	productID, err := strconv.ParseInt(req.ProductID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效商品ID")
		return
	}

	// 查询商品是否存在
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		utils.Fail(c, "商品不存在")
		return
	}

	// 检查是否被订单引用（查询订单项中是否包含该商品的任何规格）
	var count int64
	// 先查出该商品的所有规格ID
	var specIDs []int64
	database.DB.Model(&models.Spec{}).Where("product_id = ?", productID).Pluck("id", &specIDs)
	if len(specIDs) > 0 {
		database.DB.Model(&models.OrderItem{}).Where("spec_id IN ?", specIDs).Count(&count)
		if count > 0 {
			utils.Fail(c, "商品已被订单引用，无法删除")
			return
		}
	}

	// 事务删除：删除规格属性、规格、商品主表
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 删除规格属性模板
		if err := tx.Where("product_id = ?", productID).Delete(&models.SpecAttribute{}).Error; err != nil {
			return err
		}
		// 删除 SKU 规格
		if err := tx.Where("product_id = ?", productID).Delete(&models.Spec{}).Error; err != nil {
			return err
		}
		// 删除商品主表
		if err := tx.Delete(&product).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		utils.Fail(c, "删除失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}
