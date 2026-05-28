package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type FreightSetting struct {
	ID                 utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	ProvinceID         string         `gorm:"index;size:10" json:"provinceId"` // 空表示全国默认
	FirstPrice         float64        `json:"firstPrice"`                      // 首件运费（例如5元）
	AdditionalPrice    float64        `json:"additionalPrice"`                 // 续件每件运费（例如1元）
	FreeShippingAmount float64        `json:"freeShippingAmount"`              // 满额包邮（0表示不启用）
}

func (p *FreightSetting) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
