package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type ProductSpec struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	ProductID utils.Int64Str `gorm:"index;not null" json:"productId"`
	Name      string         `gorm:"size:100;not null" json:"name"` // 规格名称，如 "5寸", "6寸", "光面", "绒面"
	Image     string         `gorm:"size:255" json:"image"`         // 规格图（可选）
	Price     float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock     int            `gorm:"default:0" json:"stock"` // 库存
	SkuCode   string         `gorm:"size:50" json:"skuCode"` // 可选SKU编码
	SortOrder int            `gorm:"default:0" json:"sortOrder"`
}

func (ps *ProductSpec) BeforeCreate(tx *gorm.DB) error {
	if ps.ID == 0 {
		ps.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
