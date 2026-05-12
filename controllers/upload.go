package controllers

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// UploadPhoto 上传照片
// @Summary 上传照片
// @Description 小程序端上传打印照片
// @Tags 照片
// @Accept multipart/form-data
// @Produce json
// @Param user_id formData string true "用户ID"
// @Param file formData file true "照片文件 (jpg/png, max 5MB)"
// @Success 200 {object} utils.Response{data=models.Photo}
// @Failure 400 {object} utils.Response
// @Router /api/v1/upload [post]
func UploadPhoto(c *gin.Context) {
	userID := c.PostForm("user_id")
	if userID == "" {
		utils.Fail(c, "缺少 user_id")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		utils.Fail(c, "上传文件失败")
		return
	}
	// 限制 5MB
	if file.Size > 5<<20 {
		utils.Fail(c, "文件不能超过5MB")
		return
	}
	// 校验扩展名
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
