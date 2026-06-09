package services

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"strings"
)

// getSpecDisplayName 获取规格显示名称（从 Attributes 或 SkuKey）
func getSpecDisplayName(spec models.Spec) string {
	if len(spec.Attributes) > 0 {
		var values []string
		for _, v := range spec.Attributes {
			values = append(values, v)
		}
		return strings.Join(values, " ")
	}
	return spec.SkuKey
}

// BuildOrderSpecSummaries 基于订单项分组汇总规格信息
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
		product := spec.Product
		specID := spec.ID.Int64()
		if _, ok := group[specID]; !ok {
			group[specID] = &models.SpecSummaryResponse{
				ProductID:     product.ID.String(),
				ProductName:   product.Name,
				SpecID:        spec.ID.String(),
				SpecName:      getSpecDisplayName(spec),
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
