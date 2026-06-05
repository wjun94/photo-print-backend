package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// QRCode 二维码配置表（仅一行记录）
type QRCode struct {
	ID             utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	BusinessQRCode string          `gorm:"type:text" json:"businessQrcode"`
	GroupQRCode    string          `gorm:"type:text" json:"groupQrcode"`
	UpdatedAt      utils.LocalTime `json:"updatedAt"`
}

func (q *QRCode) BeforeCreate(tx *gorm.DB) error {
	if q.ID == 0 {
		q.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
