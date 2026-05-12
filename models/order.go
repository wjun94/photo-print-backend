package models

import (
	"time"
)

type Order struct {
	ID          uint        `gorm:"primarykey" json:"id"`
	OrderNo     string      `gorm:"uniqueIndex;size:32;not null" json:"order_no"`
	UserID      uint        `gorm:"index;not null" json:"user_id"` // 指向 wx_user.id
	Address     string      `gorm:"type:text;not null" json:"address"`
	TotalAmount float64     `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	Status      string      `gorm:"default:'pending';size:20" json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

type OrderItem struct {
	ID       uint    `gorm:"primarykey" json:"id"`
	OrderID  uint    `gorm:"index;not null" json:"order_id"`
	PhotoID  uint    `gorm:"not null" json:"photo_id"`
	Spec     string  `gorm:"size:50;not null" json:"spec"` // 如 "5寸", "6寸"
	Quantity int     `gorm:"not null" json:"quantity"`
	Price    float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	Photo    Photo   `gorm:"foreignKey:PhotoID" json:"photo,omitempty"`
}
