package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"photo-print-backend/utils"

	"gorm.io/gorm"
)

// StringArray 用于处理 JSON 数组字段，例如轮播图 URL 列表。
// 实现 Scanner 和 Valuer 接口，使 GORM 能自动将数据库 JSON 列映射为 []string。
type StringArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 数组
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StringArray: %T", value)
	}
	return json.Unmarshal(bytes, a)
}

// Value 实现 driver.Valuer 接口，将 []string 序列化为 JSON 存入数据库
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// StringMap 用于处理动态的规格属性键值对，如 {"颜色":"标准","尺寸":"5寸"}。
type StringMap map[string]string

// Scan 从数据库读取 JSON 对象
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(StringMap)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StringMap: %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value 将 map 序列化为 JSON 存入数据库
func (m StringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	return json.Marshal(m)
}

// SpecAttribute 规格属性模板，定义商品有哪些属性维度（如颜色、尺寸）以及每个维度可选的值。
// 例如：颜色 ["标准","绿色"]，尺寸 ["5寸","6寸"]。
// 前端可以据此生成规格选择器，并自动组合出所有可能的 SKU。
type SpecAttribute struct {
	ID        utils.Int64Str `gorm:"primarykey;autoIncrement:false" json:"id"` // 主键，雪花 ID
	ProductID utils.Int64Str `gorm:"index;not null" json:"-"`                  // 关联的商品 ID
	Name      string         `gorm:"size:50;not null" json:"name"`             // 属性名，如 "颜色"、"尺寸"
	Values    StringArray    `gorm:"type:json" json:"values"`                  // 可选值列表，JSON 数组，如 ["标准","绿色"]
	SortOrder int            `gorm:"default:0" json:"sortOrder"`               // 排序顺序
}

func (a *SpecAttribute) BeforeCreate(tx *gorm.DB) error {
	if a.ID == 0 {
		a.ID = utils.Int64Str(utils.NextID())
	}
	return nil
}
