package models

import (
	"time"
)

type Photo struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    string    `gorm:"index;not null" json:"user_id"` // 小程序用户标识
	ImageURL  string    `gorm:"not null" json:"image_url"`
	CreatedAt time.Time `json:"created_at"`
}
