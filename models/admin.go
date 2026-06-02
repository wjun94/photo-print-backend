package models

import (
	"photo-print-backend/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 管理员表
type Admin struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"`
	CreatedAt int64          `gorm:"autoCreateTime" json:"createdAt"`
}

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = utils.Int64Str(utils.NextID())
	}
	if a.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		a.Password = string(hashed)
	}
	return nil
}

func (a *Admin) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(password))
	return err == nil
}
