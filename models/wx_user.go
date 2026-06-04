package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// WxUser 微信用户信息模型，用于存储通过微信授权登录的用户数据
type WxUser struct {
	ID      utils.Int64Str `gorm:"primarykey" json:"id"`                        // 主键ID（自定义Int64字符串类型）
	OpenID  string         `gorm:"uniqueIndex;size:100;not null" json:"openid"` // 微信用户在当前公众号/小程序的唯一标识（不同应用中不同）
	UnionID string         `gorm:"index;size:100" json:"unionId"`               // 【关键说明】微信开放平台用户唯一标识：
	// 当用户关联到同一个微信开放平台账号时，该ID在所有绑定应用中保持一致
	// 用于跨公众号/小程序/移动应用识别同一用户（需应用已绑定开放平台）
	Nickname  string          `gorm:"size:50" json:"nickname"`          // 用户昵称
	AvatarURL string          `gorm:"size:255" json:"avatarUrl"`        // 头像URL
	Mobile    string          `gorm:"size:20;index" json:"mobile"`      // 绑定手机号（需用户授权）
	Status    int             `gorm:"default:1;index" json:"status"`    // 账号状态：1-正常 0-禁用
	InviterID utils.Int64Str  `gorm:"index;default:0" json:"inviterId"` // 邀请人ID（推广关系链），0表示无上级
	CreatedAt utils.LocalTime `json:"createdAt"`                        // 创建时间（自定义本地时间格式）
	UpdatedAt utils.LocalTime `json:"updatedAt"`                        // 更新时间
}

const (
	UserStatusNormal   = 1
	UserStatusDisabled = 0
)

func (w *WxUser) BeforeCreate(tx *gorm.DB) error {
	if w.ID == 0 {
		fullID := utils.NextID() // 完整的雪花 19 位
		// 取后 8 位数字（通过取模）
		shortID := fullID % 100_000_000 // 0 ~ 99,999,999
		// 若希望最小为 10,000,000（保证 8 位），可以加上基数
		if shortID < 10_000_000 {
			shortID += 10_000_000
		}
		w.ID = utils.Int64Str(shortID)
	}
	return nil
}
