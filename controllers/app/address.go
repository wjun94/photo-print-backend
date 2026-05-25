package app

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AddressReq struct {
	ID           string `json:"id"`
	ReceiverName string `json:"receiverName" binding:"required"`
	Mobile       string `json:"mobile" binding:"required"`
	ProvinceID   int64  `json:"provinceId" binding:"required"`
	CityID       int64  `json:"cityId" binding:"required"`
	DistrictID   int64  `json:"districtId" binding:"required"`
	Detail       string `json:"detail" binding:"required"`
	Doorplate    string `json:"doorplate"`
	IsDefault    bool   `json:"isDefault"`
}

// GetAddressList 获取当前用户的收货地址列表（分页）
func GetAddressList(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size

	var addresses []models.Address
	var total int64

	query := database.DB.Model(&models.Address{}).Where("user_id = ?", userID)
	query.Count(&total)
	query.Offset(offset).Limit(size).Order("is_default DESC, created_at DESC").Find(&addresses)

	utils.Success(c, gin.H{
		"list":  addresses,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// AddAddress 新增地址
func AddAddress(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	var req AddressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}
	// 如果设置为默认，需将其他默认清除
	if req.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	}
	addr := models.Address{
		UserID:       utils.Int64Str(userID),
		ReceiverName: req.ReceiverName,
		Mobile:       req.Mobile,
		ProvinceID:   utils.Int64Str(req.ProvinceID),
		CityID:       utils.Int64Str(req.CityID),
		DistrictID:   utils.Int64Str(req.DistrictID),
		Detail:       req.Detail,
		Doorplate:    req.Doorplate,
		IsDefault:    req.IsDefault,
	}
	if err := database.DB.Create(&addr).Error; err != nil {
		utils.Fail(c, "添加地址失败")
		return
	}
	utils.Success(c, addr)
}

// UpdateAddress 编辑地址
func UpdateAddress(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var req AddressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}
	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		utils.Fail(c, "地址不存在")
		return
	}
	// 如果设置为默认，需将其他默认清除
	if req.IsDefault && !address.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	}
	updates := map[string]interface{}{
		"receiver_name": req.ReceiverName,
		"mobile":        req.Mobile,
		"province_id":   req.ProvinceID,
		"city_id":       req.CityID,
		"district_id":   req.DistrictID,
		"detail":        req.Detail,
		"doorplate":     req.Doorplate,
		"is_default":    req.IsDefault,
	}
	if err := database.DB.Model(&address).Updates(updates).Error; err != nil {
		utils.Fail(c, "更新地址失败")
		return
	}
	utils.Success(c, nil)
}

// DeleteAddress 删除地址
func DeleteAddress(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Address{})
	if result.RowsAffected == 0 {
		utils.Fail(c, "地址不存在")
		return
	}
	utils.Success(c, nil)
}

// SetDefaultAddress 设置默认地址
func SetDefaultAddress(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	// 清除其他默认
	if err := database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
		utils.Fail(c, "设置失败")
		return
	}
	result := database.DB.Model(&models.Address{}).Where("id = ? AND user_id = ?", id, userID).Update("is_default", true)
	if result.RowsAffected == 0 {
		utils.Fail(c, "地址不存在")
		return
	}
	utils.Success(c, nil)
}

// GetAddressDetail 获取单个地址详情
func GetAddressDetail(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}
	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		utils.Fail(c, "地址不存在")
		return
	}
	utils.Success(c, address)
}
