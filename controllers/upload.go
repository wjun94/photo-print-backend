package controllers

import (
	"photo-print-backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func UploadPhoto(c *gin.Context) {
	// 使用辅助函数获取用户ID（int64）
	_, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录或用户ID无效")
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
	utils.Success(c, gin.H{
		"url": imageURL, // 这里直接映射成 url 字段
	})
}
