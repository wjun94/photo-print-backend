package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

const (
	OrderStatusPending   = "pending"   // 待付款
	OrderStatusPaid      = "paid"      // 已付款/待发货
	OrderStatusShipped   = "shipped"   // 已发货
	OrderStatusCompleted = "completed" // 已完成
	OrderStatusCancelled = "cancelled" // 已取消
	OrderStatusRefunding = "refunding" // 退款中
	OrderStatusRefunded  = "refunded"  // 已退款
)

// SpecSummaryResponse 规格汇总响应（不重复）
type SpecSummaryResponse struct {
	ProductID     string  `json:"productId"`
	ProductName   string  `json:"productName"`
	SpecID        string  `json:"specId"`
	SpecName      string  `json:"specName"`
	Price         float64 `json:"price"`
	TotalQuantity int     `json:"totalQuantity"` // 该规格的总数量
	TotalSubtotal float64 `json:"totalSubtotal"` // 该规格的总小计
	ImageURL      string  `json:"imageUrl"`
}

type Order struct {
	ID        utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	OrderNo   string          `gorm:"uniqueIndex;size:32;not null" json:"orderNo"`
	UserID    utils.Int64Str  `gorm:"index;not null" json:"userId"` // 指向 wx_user.id
	Status    string          `gorm:"default:'pending';size:20" json:"status"`
	CreatedAt utils.LocalTime `json:"createdAt"`
	UpdatedAt utils.LocalTime `json:"updatedAt"`
	// 各阶段时间（指针类型可为空）
	PayAt     *utils.LocalTime `gorm:"type:datetime" json:"payTime,omitempty"`    // 支付时间
	FinishAt  *utils.LocalTime `gorm:"type:datetime" json:"finishTime,omitempty"` // 订单完成时间
	ShippedAt *utils.LocalTime `gorm:"type:datetime" json:"shippedAt,omitempty"`  // 订单发货时间
	CancelAt  *utils.LocalTime `gorm:"type:datetime" json:"cancelTime,omitempty"` // 订单取消时间

	Remark       string                `json:"remark"`                                               // 可选备注
	Amount       float64               `gorm:"type:decimal(10,2);not null" json:"amount"`            // 商品总额
	Freight      float64               `gorm:"type:decimal(10,2);not null;default:0" json:"freight"` // 运费
	ActualAmount float64               `gorm:"type:decimal(10,2);not null" json:"actualAmount"`      // 实付 = amount + freight
	Items        []OrderItem           `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Specs        []SpecSummaryResponse `gorm:"-" json:"specs,omitempty"`
	// 新增物流关联（一个订单多个包裹）
	Logistics []Logistics  `gorm:"foreignKey:OrderID" json:"logistics,omitempty"`
	Address   OrderAddress `gorm:"foreignKey:OrderID;references:ID" json:"address"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == 0 {
		o.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
