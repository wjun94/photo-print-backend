package models

import (
	"time"
)

type WxUser struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	OpenID    string    `gorm:"uniqueIndex;size:100;not null" json:"open_id"`
	UnionID   string    `gorm:"index;size:100" json:"union_id"`
	Nickname  string    `gorm:"size:50" json:"nickname"`
	AvatarURL string    `gorm:"size:255" json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
