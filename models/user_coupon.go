package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

// UserCouponStatus 用户持有优惠券状态枚举
type UserCouponStatus int

const (
	UserCouponUnused   UserCouponStatus = 0 // 未使用：正常可抵扣
	UserCouponUsed     UserCouponStatus = 1 // 已使用：下单核销完成
	UserCouponExpired  UserCouponStatus = 2 // 已过期：超出有效期无法使用
	UserCouponCanceled UserCouponStatus = 3 // 已作废：订单退款/取消后券退回作废不可再用
)

// UserCoupon 用户优惠券关联表
// 记录每个用户领取到的单张优惠券实例，存储单张券的有效期、核销信息、归属用户
type UserCoupon struct {
	ID         utils.Int64Str   `gorm:"primarykey;autoIncrement:false" json:"id"` // 雪花唯一主键ID
	UserID     utils.Int64Str   `gorm:"index;not null" json:"userId"`             // 所属用户ID，建立索引快速查询用户全部券
	CouponID   utils.Int64Str   `gorm:"index;not null" json:"couponId"`           // 关联优惠券模板ID
	Status     UserCouponStatus `gorm:"default:0;not null" json:"status"`         // 券当前状态，对应UserCouponStatus枚举
	OrderID    utils.Int64Str   `json:"orderId,omitempty"`                        // 核销订单ID，未使用时为空
	UsedAt     *time.Time       `json:"usedAt,omitempty"`                         // 核销使用时间，未使用为nil
	ReceivedAt time.Time        `json:"receivedAt"`                               // 用户领取该券的时间
	ValidStart time.Time        `json:"validStart"`                               // 单张券生效起始时间
	ValidEnd   time.Time        `json:"validEnd"`                                 // 单张券失效截止时间
}

// BeforeCreate GORM创建前钩子函数
// 新增用户优惠券时自动生成雪花ID，不依赖数据库自增
func (uc *UserCoupon) BeforeCreate(tx *gorm.DB) error {
	if uc.ID == 0 {
		uc.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
