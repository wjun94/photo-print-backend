package common

import (
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

// CourierCompany 快递公司结构
type CourierCompany struct {
	Code string `json:"code"` // 编码，如 SF
	Name string `json:"name"` // 名称，如 顺丰速运
}

// GetCourierList 获取快递公司列表（静态数据）
// @Summary 快递公司列表
// @Description 返回常用快递公司的编码和名称
// @Tags 公共接口
// @Produce json
// @Success 200 {object} utils.Response{data=[]CourierCompany}
// @Router /api/v1/couriers [get]
func GetCourierList(c *gin.Context) {
	// 静态数据，可在此增删改，或从配置文件读取
	couriers := []CourierCompany{
		{Code: "SF", Name: "顺丰速运"},
		{Code: "YTO", Name: "圆通速递"},
		{Code: "ZTO", Name: "中通快递"},
		{Code: "STO", Name: "申通快递"},
		{Code: "YD", Name: "韵达快递"},
		{Code: "EMS", Name: "中国邮政"},
		{Code: "JD", Name: "京东物流"},
		{Code: "DB", Name: "德邦快递"},
		{Code: "JT", Name: "极兔速递"},
		{Code: "CAE", Name: "菜鸟裹裹"},
	}
	utils.Success(c, couriers)
}
