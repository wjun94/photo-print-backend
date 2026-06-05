package common

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
)

// GetQRCodes 获取二维码地址（公开接口）
func GetQRCodes(c *gin.Context) {
	var qr models.QRCode
	err := database.DB.First(&qr).Error
	if err != nil {
		// 若不存在，返回空字符串
		utils.Success(c, gin.H{
			"businessQrcode": "",
			"groupQrcode":    "",
		})
		return
	}
	utils.Success(c, gin.H{
		"businessQrcode": qr.BusinessQRCode,
		"groupQrcode":    qr.GroupQRCode,
	})
}
