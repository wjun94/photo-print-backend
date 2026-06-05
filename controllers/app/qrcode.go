package app

import (
	"photo-print-backend/controllers/common"

	"github.com/gin-gonic/gin"
)

// GetQRCodes 获取商务合作和交流群二维码URL
// @Summary 获取二维码图片地址
// @Description 返回商务合作二维码和交流群二维码的URL
// @Tags 公共接口
// @Produce json
// @Success 200 {object} utils.Response{data=object{businessQrcode=string,groupQrcode=string}}
// @Router /api/v1/qrcodes [get]
func GetQRCodes(c *gin.Context) {
	// 复用公共逻辑
	common.GetQRCodes(c)
}
