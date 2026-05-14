package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID          int64       `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderNo     string      `gorm:"uniqueIndex;size:32;not null" json:"order_no"`
	UserID      int64       `gorm:"index;not null" json:"user_id"` // 指向 wx_user.id
	Address     string      `gorm:"type:text;not null" json:"address"`
	TotalAmount float64     `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	Status      string      `gorm:"default:'pending';size:20" json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == 0 {
		o.ID = utils.NextID()
	}
	return nil
}

type OrderItem struct {
	ID       utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderID  utils.Int64Str `gorm:"index;not null" json:"order_id"`
	PhotoID  uint           `gorm:"not null" json:"photo_id"`
	Spec     string         `gorm:"size:50;not null" json:"spec"` // 如 "5寸", "6寸"
	Quantity int            `gorm:"not null" json:"quantity"`
	Price    float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	Photo    Photo          `gorm:"foreignKey:PhotoID" json:"photo,omitempty"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == 0 {
		oi.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
