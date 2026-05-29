package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// 商品规格
type OrderItem struct {
	ID       utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderID  utils.Int64Str `gorm:"index;not null" json:"orderId"`
	Spec     string         `gorm:"size:50;not null" json:"spec"` // 如 "5寸", "6寸"
	SpecID   utils.Int64Str `gorm:"index;not null" json:"specId"` // 新增：关联 product_specs.id
	Quantity int            `gorm:"not null" json:"quantity"`
	Price    float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	ImageURL string         `gorm:"type:text;not null" json:"imageUrl"`
	SpecInfo ProductSpec    `gorm:"foreignKey:SpecID" json:"-"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == 0 {
		oi.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
