package models

import (
	"photo-print-backend/utils"
	"time"

	"gorm.io/gorm"
)

// CouponType 优惠券类型枚举
type CouponType int

const (
	CouponTypeFullReduce  CouponType = 1 // 满减券：满足指定金额减免固定金额
	CouponTypeNoThreshold CouponType = 2 // 无门槛券：无消费门槛直接抵扣
	CouponTypeDiscount    CouponType = 3 // 折扣券：按比例打折，可设置最高减免上限
)

// PublishStatus 优惠券发布上下架状态
type PublishStatus int

const (
	PublishStatusUnpublished PublishStatus = 0 // 未发布：草稿状态，用户不可领取
	PublishStatusPublished   PublishStatus = 1 // 已发布：正常上架，可领取使用
	PublishStatusOffline     PublishStatus = 2 // 已下架：手动下架，停止发放
)

// UseScope 优惠券使用商品范围
type UseScope int

const (
	UseScopeAll  UseScope = 1 // 全平台通用：店内所有商品均可使用
	UseScopeSpec UseScope = 2 // 指定商品：仅配置的商品可使用
)

// TimeType 优惠券有效期类型
type TimeType int

const (
	TimeTypeFixed TimeType = 1 // 固定时间段：指定起止日期，统一失效
	TimeTypeAfter TimeType = 2 // 领取后N天有效：用户领券开始倒计时有效期
)

// Coupon 优惠券主表模型
// 存储平台所有优惠券配置、库存、发放规则、使用门槛等基础信息
type Coupon struct {
	ID             utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`         // 雪花算法唯一主键ID
	Name           string         `gorm:"size:128;not null" json:"name"`                    // 优惠券名称
	Type           CouponType     `gorm:"not null" json:"type"`                             // 优惠券类型，对应CouponType枚举
	PublishStatus  PublishStatus  `gorm:"default:0;not null" json:"publishStatus"`          // 发布状态，对应PublishStatus枚举
	FullAmount     float64        `gorm:"type:decimal(10,2);default:0" json:"fullAmount"`   // 满减门槛金额，满减券生效
	ReduceAmount   float64        `gorm:"type:decimal(10,2);default:0" json:"reduceAmount"` // 满减抵扣金额
	DiscountRate   float64        `gorm:"type:decimal(3,2);default:1" json:"discountRate"`  // 折扣比例 0.8=8折，折扣券生效
	MaxReduce      float64        `gorm:"type:decimal(10,2);default:0" json:"maxReduce"`    // 折扣券最高减免金额，0代表无上限
	UseScope       UseScope       `gorm:"not null" json:"useScope"`                         // 使用商品范围枚举
	ProductIds     string         `gorm:"type:text" json:"productIds"`                      // 指定商品ID列表，多个用英文逗号分隔，UseScope=2时生效
	TimeType       TimeType       `gorm:"not null" json:"timeType"`                         // 有效期类型枚举
	ValidStart     *time.Time     `json:"validStart,omitempty"`                             // 固定有效期-开始时间，TimeType=1必填
	ValidEnd       *time.Time     `json:"validEnd,omitempty"`                               // 固定有效期-结束时间，TimeType=1必填
	ValidDays      int            `gorm:"default:0" json:"validDays"`                       // 领券后有效天数，TimeType=2必填
	TotalStock     int            `gorm:"default:0;not null" json:"totalStock"`             // 优惠券总发放库存
	ReceivedNum    int            `gorm:"default:0;not null" json:"receivedNum"`            // 已被用户领取数量
	UserLimitType  int            `gorm:"not null" json:"userLimitType"`                    // 用户领券限制类型：1不限、2单人限N张、3单人限1张
	UserLimitNum   int            `gorm:"default:1" json:"userLimitNum"`                    // 单人最多领取数量，UserLimitType=2生效
	TargetUserType int            `gorm:"default:0" json:"targetUserType"`                  // 目标用户类型 0全部用户、其他值可自定义新用户/老用户等
	ReceiveStart   time.Time      `gorm:"not null" json:"receiveStart"`                     // 领券活动开始时间
	ReceiveEnd     time.Time      `gorm:"not null" json:"receiveEnd"`                       // 领券活动结束时间
	Desc           string         `gorm:"type:text" json:"desc"`                            // 优惠券使用说明、活动描述
	CreatedAt      time.Time      `json:"createdAt"`                                        // 创建时间
	UpdatedAt      time.Time      `json:"updatedAt"`                                        // 更新时间
}

// BeforeCreate GORM创建前钩子
// 自动生成雪花ID，替代数据库自增主键
func (c *Coupon) BeforeCreate(tx *gorm.DB) error {
	if c.ID == 0 {
		c.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
