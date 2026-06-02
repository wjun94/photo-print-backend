package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// CommissionSetting 返利比例配置（历史记录）
type CommissionSetting struct {
	ID        utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	Ratio     float64         `gorm:"type:decimal(5,2);not null;default:10.00" json:"ratio"` // 返利百分比（如 10.00 表示 10%）
	UpdatedBy string          `gorm:"size:100" json:"updatedBy"`                             // 操作人
	UpdatedAt utils.LocalTime `json:"updatedAt"`
}

func (c *CommissionSetting) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
