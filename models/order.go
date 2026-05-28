package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// SpecSummaryResponse 规格汇总响应（不重复）
type SpecSummaryResponse struct {
	ProductID     string  `json:"productId"`
	ProductName   string  `json:"productName"`
	SpecID        string  `json:"specId"`
	SpecName      string  `json:"specName"`
	Price         float64 `json:"price"`
	TotalQuantity int     `json:"totalQuantity"` // 该规格的总数量
	TotalSubtotal float64 `json:"totalSubtotal"` // 该规格的总小计
	ImageURL      string  `json:"imageUrl"`
}

type Order struct {
	ID           utils.Int64Str        `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderNo      string                `gorm:"uniqueIndex;size:32;not null" json:"orderNo"`
	UserID       utils.Int64Str        `gorm:"index;not null" json:"userId"` // 指向 wx_user.id
	Address      string                `gorm:"type:text;not null" json:"address"`
	Status       string                `gorm:"default:'pending';size:20" json:"status"`
	CreatedAt    utils.LocalTime       `json:"createdAt"`
	UpdatedAt    utils.LocalTime       `json:"updatedAt"`
	Remark       string                `json:"remark"`                                               // 可选备注
	Amount       float64               `gorm:"type:decimal(10,2);not null" json:"amount"`            // 商品总额
	Freight      float64               `gorm:"type:decimal(10,2);not null;default:0" json:"freight"` // 运费
	ActualAmount float64               `gorm:"type:decimal(10,2);not null" json:"actualAmount"`      // 实付 = amount + freight
	Items        []OrderItem           `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Specs        []SpecSummaryResponse `gorm:"-" json:"specs"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == 0 {
		o.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}

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
