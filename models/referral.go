package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// 推广关系表
type Referral struct {
	ID        utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	UserID    utils.Int64Str  `gorm:"index;not null" json:"userId"`         // 推广者（上级）
	FriendID  utils.Int64Str  `gorm:"uniqueIndex;not null" json:"friendId"` // 被邀请的好友（下级）
	CreatedAt utils.LocalTime `json:"createdAt"`
}

func (r *Referral) BeforeCreate(tx *gorm.DB) error {
	if r.ID == 0 {
		r.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
