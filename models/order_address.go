package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// OrderAddress 订单地址快照表
// 重要：不关联用户地址簿ID，所有信息完整复制，确保历史订单地址永久不变
type OrderAddress struct {
	ID           utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderID      utils.Int64Str  `gorm:"uniqueIndex:idx_order_id;not null" json:"orderId"` // 一对一关联订单
	ReceiverName string          `gorm:"size:50;not null" json:"receiverName"`
	Mobile       string          `gorm:"size:20;not null" json:"mobile"`
	ProvinceID   string          `gorm:"size:20;not null" json:"provinceId"`
	ProvinceName string          `gorm:"size:50;not null" json:"provinceName"` // 必须存名称！防止省市区字典表修改
	CityID       string          `gorm:"size:20;not null" json:"cityId"`
	CityName     string          `gorm:"size:50;not null" json:"cityName"`
	DistrictID   string          `gorm:"size:20;not null" json:"districtId"`
	DistrictName string          `gorm:"size:50;not null" json:"districtName"`
	Detail       string          `gorm:"type:text;not null" json:"detail"`
	Doorplate    string          `gorm:"size:100" json:"doorplate"` // 门牌号（选填）
	CreatedAt    utils.LocalTime `json:"createdAt"`
	UpdatedAt    utils.LocalTime `json:"updatedAt"`
}

func (oi *OrderAddress) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == 0 {
		oi.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
