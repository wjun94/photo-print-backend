package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type Logistics struct {
	ID          utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderID     utils.Int64Str  `gorm:"index:idx_order_id;not null" json:"orderId"`               // 关联订单
	TrackingNo  string          `gorm:"index:idx_tracking_no;size:50;not null" json:"trackingNo"` // 物流单号
	CourierCode string          `gorm:"size:20;not null" json:"courierCode"`                      // 快递公司编码（如：SF、YTO、ZTO）
	CourierName string          `gorm:"size:50;not null" json:"courierName"`                      // 快递公司名称（必须存，防止字典表修改）
	Remark      string          `json:"remark"`                                                   // 可选备注
	CreatedAt   utils.LocalTime `json:"createdAt"`
	UpdatedAt   utils.LocalTime `json:"updatedAt"`
}

func (o *Logistics) BeforeCreate(tx *gorm.DB) error {
	if o.ID == 0 {
		o.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
