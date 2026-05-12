package controllers

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetPhotoList 照片列表 (后台)
// @Summary 照片列表
// @Description 后台查看所有上传的照片
// @Tags 照片
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {object} utils.Response{data=[]models.Photo}
// @Router /api/v1/photos [get]
func GetPhotoList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size
	var photos []models.Photo
	var total int64
	database.DB.Model(&models.Photo{}).Count(&total)
	database.DB.Offset(offset).Limit(size).Order("created_at desc").Find(&photos)
	utils.Success(c, gin.H{
		"list":  photos,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
