package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Username string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password string `gorm:"not null" json:"-"` // 不返回到前端
}

// 创建用户前自动哈希密码
func (u *User) BeforeCreate(tx *gorm.DB) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

// 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
