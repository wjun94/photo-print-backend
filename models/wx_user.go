package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type WxUser struct {
	ID        utils.Int64Str  `gorm:"primarykey" json:"id"`
	OpenID    string          `gorm:"uniqueIndex;size:100;not null" json:"openId"`
	UnionID   string          `gorm:"index;size:100" json:"unionId"`
	Nickname  string          `gorm:"size:50" json:"nickname"`
	AvatarURL string          `gorm:"size:255" json:"avatarUrl"`
	Mobile    string          `gorm:"size:20;index" json:"mobile"`
	Status    int             `gorm:"default:1;index" json:"status"` // 1:正常 0:禁用
	CreatedAt utils.LocalTime `json:"createdAt"`
	UpdatedAt utils.LocalTime `json:"updatedAt"`
}

const (
	UserStatusNormal   = 1
	UserStatusDisabled = 0
)

func (w *WxUser) BeforeCreate(tx *gorm.DB) error {
	if w.ID == 0 {
		fullID := utils.NextID() // 完整的雪花 19 位
		// 取后 8 位数字（通过取模）
		shortID := fullID % 100_000_000 // 0 ~ 99,999,999
		// 若希望最小为 10,000,000（保证 8 位），可以加上基数
		if shortID < 10_000_000 {
			shortID += 10_000_000
		}
		w.ID = utils.Int64Str(shortID)
	}
	return nil
}
