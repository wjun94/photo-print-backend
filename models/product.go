package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

type ProductStatus string

const (
	ProductStatusDraft   ProductStatus = "draft"    // 草稿
	ProductStatusOnSale  ProductStatus = "on_sale"  // 上架
	ProductStatusOffSale ProductStatus = "off_sale" // 下架
)

type Product struct {
	ID           utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"`
	Name         string          `gorm:"size:200;not null" json:"name"`
	Description  string          `gorm:"type:text" json:"description"`  // 简短描述
	Detail       string          `gorm:"type:longtext" json:"detail"`   // 富文本详情（HTML/Markdown）
	CoverImage   string          `gorm:"size:500" json:"coverImage"`    // 封面图（单张）
	BannerImages []string        `gorm:"type:json" json:"bannerImages"` // 轮播图数组
	Status       ProductStatus   `gorm:"default:'draft';size:20" json:"status"`
	SortOrder    int             `gorm:"default:0" json:"sortOrder"` // 排序
	CreatedAt    utils.LocalTime `json:"createdAt"`
	UpdatedAt    utils.LocalTime `json:"updatedAt"`
	Specs        []ProductSpec   `gorm:"foreignKey:ProductID" json:"specs,omitempty"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
