package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Admin struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	Username  string `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string `gorm:"not null" json:"-"`
	CreatedAt int64  `gorm:"autoCreateTime" json:"created_at"`
}

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
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
