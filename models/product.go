package models

import (
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// ProductStatus 商品状态枚举
type ProductStatus string

const (
	ProductStatusDraft   ProductStatus = "draft"    // 草稿
	ProductStatusOnSale  ProductStatus = "on_sale"  // 上架
	ProductStatusOffSale ProductStatus = "off_sale" // 下架
)

type ProductAction string

const (
	ProductActionConfirm ProductAction = "confirm" // 前往确认订单页
	ProductActionUpload  ProductAction = "upload"  // 前往上传照片页
)

// Product 商品主表，包含基本信息、多规格属性和 SKU 列表。
type Product struct {
	ID           utils.Int64Str  `gorm:"primarykey;autoIncrement:false" json:"id"` // 商品 ID，雪花算法生成
	Name         string          `gorm:"size:200;not null" json:"name"`            // 商品名称
	CoverImage   string          `gorm:"size:500" json:"coverImage"`               // 封面图片 URL
	BannerImages StringArray     `gorm:"type:json" json:"bannerImages"`            // 轮播图 URL 数组，JSON 存储
	Description  string          `gorm:"type:text" json:"description"`             // 简短描述，用于列表页
	Detail       string          `gorm:"type:longtext" json:"detail"`              // 富文本详情（HTML/Markdown）
	Status       ProductStatus   `gorm:"default:'draft';size:20" json:"status"`    // 商品状态：draft, on_sale, off_sale
	SortOrder    int             `gorm:"default:0" json:"sortOrder"`               // 排序序号，数字越小越靠前
	CreatedAt    utils.LocalTime `json:"createdAt"`                                // 创建时间
	UpdatedAt    utils.LocalTime `json:"updatedAt"`                                // 更新时间
	Action       ProductAction   `gorm:"size:20;default:'confirm'" json:"action"`  // confirm-确认订单, upload-上传照片

	// 关联的规格属性模板（如颜色、尺寸的可选值）
	SpecAttributes []SpecAttribute `gorm:"foreignKey:ProductID" json:"specAttributes,omitempty"`
	// 具体的 SKU 规格列表（每个规格包含价格、库存及属性组合）
	Specs []Spec `gorm:"foreignKey:ProductID" json:"specs,omitempty"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == 0 {
		p.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
