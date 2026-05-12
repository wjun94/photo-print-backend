package controllers

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func UploadPhoto(c *gin.Context) {
	// 获取当前登录用户ID（小程序用户）
	userIDVal, exists := c.Get("user_id")
	if !exists {
		utils.Fail(c, "未登录")
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		utils.Fail(c, "用户身份无效")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		utils.Fail(c, "上传文件失败")
		return
	}
	// 限制5MB
	if file.Size > 5<<20 {
		utils.Fail(c, "文件不能超过5MB")
		return
	}
	ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, "."):])
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		utils.Fail(c, "只支持 jpg, jpeg, png 格式")
		return
	}

	uploadDir := "./uploads"
	savedName, err := utils.SaveUploadedFile(file, uploadDir)
	if err != nil {
		utils.Fail(c, "保存文件失败")
		return
	}
	imageURL := "/uploads/" + savedName
	photo := models.Photo{
		UserID:   userID,
		ImageURL: imageURL,
	}
	if err := database.DB.Create(&photo).Error; err != nil {
		utils.Fail(c, "保存记录失败")
		return
	}
	utils.Success(c, photo)
}
