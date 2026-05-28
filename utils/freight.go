package utils

import (
	"photo-print-backend/config"
)

// CalculateFreight 根据商品总额计算运费
func CalculateFreight(totalAmount float64) float64 {
	cfg := config.AppConfig
	if cfg.FreeShippingAmount > 0 && totalAmount >= cfg.FreeShippingAmount {
		return 0
	}
	return cfg.FixedFreight
}

// 更复杂的运费规则（按省份差异化）暂时用不到
/* func CalculateFreightByProvince(provinceID string, totalAmount float64, totalQuantity int) float64 {
	// 查询省份运费规则
	var setting models.FreightSetting
	err := database.DB.Where("province_id = ? OR province_id = ''", provinceID).Order("province_id DESC").First(&setting).Error
	if err != nil {
		// 默认运费
		return config.AppConfig.FixedFreight
	}
	// 满额包邮
	if setting.FreeShippingAmount > 0 && totalAmount >= setting.FreeShippingAmount {
		return 0
	}
	if setting.FirstPrice == 0 {
		return 0
	}
	// 首件+续件
	if totalQuantity <= 1 {
		return setting.FirstPrice
	}
	additional := float64(totalQuantity-1) * setting.AdditionalPrice
	return setting.FirstPrice + additional
} */
