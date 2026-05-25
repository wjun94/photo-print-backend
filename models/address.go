package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type Address struct {
	ID           utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	UserID       utils.Int64Str  `gorm:"index;not null" json:"userId"`
	ReceiverName string          `gorm:"size:50;not null" json:"receiverName"`
	Mobile       string          `gorm:"size:20;not null" json:"mobile"`
	ProvinceID   string          `gorm:"index;not null" json:"provinceId"`
	CityID       string          `gorm:"index;not null" json:"cityId"`
	DistrictID   string          `gorm:"index;not null" json:"districtId"`
	Detail       string          `gorm:"type:text;not null" json:"detail"`
	Doorplate    string          `gorm:"size:100" json:"doorplate"` // 门牌号（选填）
	IsDefault    bool            `gorm:"default:false" json:"isDefault"`
	CreatedAt    utils.LocalTime `json:"createdAt"`
	UpdatedAt    utils.LocalTime `json:"updatedAt"`
}

func (a *Address) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
