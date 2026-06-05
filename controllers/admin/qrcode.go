package admin

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"photo-print-backend/config"
	"photo-print-backend/controllers/common"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadQRCodes 上传二维码图片（支持部分更新）
// @Summary 上传二维码图片
// @Description 上传商务合作二维码和/或交流群二维码图片，不传的字段保留原值
// @Tags 配置管理
// @Accept multipart/form-data
// @Produce json
// @Param business formData file false "商务合作二维码图片"
// @Param group formData file false "交流群二维码图片"
// @Success 200 {object} utils.Response{data=object{businessQrcode=string,groupQrcode=string}}
// @Failure 400 {object} utils.Response
// @Router /api/v1/admin/qrcode/upload [post]
func UploadQRCodes(c *gin.Context) {
	// 获取现有记录
	var qr models.QRCode
	result := database.DB.First(&qr)
	if result.Error != nil {
		// 初始化空记录（如果表里还没有数据）
		qr = models.QRCode{}
	}

	businessFile, _ := c.FormFile("business")
	groupFile, _ := c.FormFile("group")

	// 辅助上传函数
	uploadOne := func(file *multipart.FileHeader, prefix string) (string, error) {
		if file == nil {
			return "", nil // 没有文件不处理
		}
		ext := filepath.Ext(file.Filename)
		fileName := fmt.Sprintf("%s/%d%s", prefix, time.Now().UnixNano(), ext)
		tempPath := filepath.Join("/tmp", fileName)
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			return "", err
		}
		defer os.Remove(tempPath)
		url, err := utils.UploadToQiniu(fileName, tempPath)
		if err != nil {
			return "", err
		}
		return url, nil
	}

	// 处理商务二维码
	if businessFile != nil {
		url, err := uploadOne(businessFile, config.AppConfig.UploadPrefixQrcode)
		if err != nil {
			utils.Fail(c, "上传商务二维码失败: "+err.Error())
			return
		}
		qr.BusinessQRCode = url
	}

	// 处理群二维码
	if groupFile != nil {
		url, err := uploadOne(groupFile, config.AppConfig.UploadPrefixQrcode)
		if err != nil {
			utils.Fail(c, "上传交流群二维码失败: "+err.Error())
			return
		}
		qr.GroupQRCode = url
	}

	// 如果没有传入任何文件，返回当前值（也可直接返回）
	if businessFile == nil && groupFile == nil {
		utils.Success(c, gin.H{
			"businessQrcode": qr.BusinessQRCode,
			"groupQrcode":    qr.GroupQRCode,
		})
		return
	}

	now := utils.LocalTime(time.Now())
	qr.UpdatedAt = now
	if result.Error != nil {
		database.DB.Create(&qr)
	} else {
		database.DB.Save(&qr)
	}

	utils.Success(c, gin.H{
		"businessQrcode": qr.BusinessQRCode,
		"groupQrcode":    qr.GroupQRCode,
	})
}

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
