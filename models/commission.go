package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

const (
	CommissionStatusPending = "pending" // 待结算
	CommissionStatusSettled = "settled" // 已结算
)

// Commission 佣金记录表
type Commission struct {
	ID          utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	UserID      utils.Int64Str  `gorm:"index;not null" json:"userId"`                   // 推广者ID（上级）
	FriendID    utils.Int64Str  `gorm:"index;not null" json:"friendId"`                 // 下单用户ID（下级）
	OrderID     utils.Int64Str  `gorm:"uniqueIndex;not null" json:"orderId"`            // 订单ID（唯一防重）
	OrderAmount float64         `gorm:"type:decimal(10,2);not null" json:"orderAmount"` // 订单实付金额
	Ratio       float64         `gorm:"type:decimal(5,2);not null" json:"ratio"`        // 当时返利比例
	Amount      float64         `gorm:"type:decimal(10,2);not null" json:"amount"`      // 佣金金额
	Status      string          `gorm:"size:20;default:'pending'" json:"status"`        // pending / settled
	CreatedAt   utils.LocalTime `json:"createdAt"`
}

func (c *Commission) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
