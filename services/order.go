package services

import (
	"fmt"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"sort"
	"strings"
)

// getSpecDisplayName 获取规格显示名称（从 Attributes 拼接或使用 SkuKey）
func GetSpecDisplayName(spec models.Spec) string {
	if len(spec.Attributes) > 0 {
		// 对属性名排序保证顺序稳定
		keys := make([]string, 0, len(spec.Attributes))
		for k := range spec.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s:%s", k, spec.Attributes[k]))
		}
		return strings.Join(parts, " ")
	}
	// 如果 Attributes 为空，则使用 SkuKey 并替换下划线或逗号为空格
	display := strings.ReplaceAll(spec.SkuKey, "_", " ")
	display = strings.ReplaceAll(display, ",", " ")
	return display
}

// BuildOrderSpecSummaries 构建订单规格汇总
func BuildOrderSpecSummaries(orderID int64) ([]models.SpecSummaryResponse, error) {
	var items []models.OrderItem
	err := database.DB.
		Where("order_id = ?", orderID).
		Preload("SpecInfo.Product").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	group := make(map[int64]*models.SpecSummaryResponse)
	for _, it := range items {
		spec := it.SpecInfo
		// 如果规格不存在，跳过（数据库可能数据异常）
		if spec.ID == 0 {
			continue
		}
		product := spec.Product
		specID := spec.ID.Int64()
		if _, ok := group[specID]; !ok {
			group[specID] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        spec.ID.String(),
				SpecName:      GetSpecDisplayName(spec),
				Price:         it.Price,
				TotalQuantity: 0,
				TotalSubtotal: 0,
				ImageURL:      product.CoverImage,
			}
		}
		group[specID].TotalQuantity += it.Quantity
		group[specID].TotalSubtotal += float64(it.Quantity) * it.Price
	}

	summaries := make([]models.SpecSummaryResponse, 0, len(group))
	for _, v := range group {
		summaries = append(summaries, *v)
	}
	return summaries, nil
}
