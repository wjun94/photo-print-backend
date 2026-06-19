package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// Spec 具体的 SKU（库存量单位）规格。
// 每个 Spec 代表一种确定的规格组合，包含价格、库存、SKU 编码等。
// 例如：颜色=标准 + 尺寸=5寸 对应一条 Spec 记录。
type Spec struct {
	ID utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"` // 主键
	// 1. 将 ProductID 变更为联合唯一索引的一部分，命名为 idx_product_sku，使其与 SkuKey 组合成联合唯一索引，确保同一商品下的规格唯一性
	ProductID utils.Int64Str `gorm:"uniqueIndex:idx_product_sku;not null" json:"-"` // 所属商品 ID
	// 2. 给 SkuKey 的 uniqueIndex 命名，使其与 ProductID 组合成联合唯一索引 idx_product_sku
	SkuKey     string    `gorm:"uniqueIndex:idx_product_sku;size:200" json:"skuKey"` // 规格组合的唯一键，如 "颜色:标准_尺寸:5寸"，用于前端快速匹配。idx_product_sku不同商品可以使用相同的skuKey
	Price      float64   `gorm:"type:decimal(10,2);not null" json:"price"`           // 该规格的售价
	Stock      int       `gorm:"default:0" json:"stock"`                             // 库存数量
	SkuCode    string    `gorm:"size:50" json:"skuCode"`                             // 商家自定义 SKU 编码
	Image      string    `gorm:"size:255" json:"image"`                              // 规格图（可覆盖商品主图）
	Attributes StringMap `gorm:"type:json;serializer:json" json:"attributes"`        // 动态属性，存储键值对如 {"颜色":"标准","尺寸":"5寸"}
	// Product    Product        `gorm:"foreignKey:ProductID" json:"-"`
}

func (s *Spec) BeforeCreate(tx *gorm.DB) error {
	if s.ID == 0 {
		s.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
