package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderNo   string         `gorm:"uniqueIndex;size:32;not null" json:"orderNo"`
	UserID    utils.Int64Str `gorm:"index;not null" json:"userId"` // 指向 wx_user.id
	Address   string         `gorm:"type:text;not null" json:"address"`
	Amount    float64        `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status    string         `gorm:"default:'pending';size:20" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Items     []OrderItem    `gorm:"foreignKey:OrderID" json:"items,omitempty"`
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
	PhotoID  uint           `gorm:"not null" json:"photoId"`
	Spec     string         `gorm:"size:50;not null" json:"spec"` // 如 "5寸", "6寸"
	Quantity int            `gorm:"not null" json:"quantity"`
	Price    float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	ImageURL string         `gorm:"type:text;not null" json:"imageUrl"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == 0 {
		oi.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
