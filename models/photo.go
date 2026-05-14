package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

type Photo struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	UserID    utils.Int64Str `gorm:"index;not null" json:"user_id"` // 指向 wx_user.id
	ImageURL  string         `gorm:"not null" json:"image_url"`
	CreatedAt time.Time      `json:"created_at"`
}

func (p *Photo) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
