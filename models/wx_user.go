package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

type WxUser struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	OpenID    string         `gorm:"uniqueIndex;size:100;not null" json:"openId"`
	UnionID   string         `gorm:"index;size:100" json:"unionId"`
	Nickname  string         `gorm:"size:50" json:"nickname"`
	AvatarURL string         `gorm:"size:255" json:"avatarUrl"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

func (w *WxUser) BeforeCreate(tx *gorm.DB) error {
	if w.ID == 0 {
		w.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
