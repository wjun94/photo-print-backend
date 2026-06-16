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
	ProvinceID   string `json:"provinceId" binding:"required"`
	CityID       string `json:"cityId" binding:"required"`
	DistrictID   string `json:"districtId" binding:"required"`
	Detail       string `json:"detail" binding:"required"`
	Doorplate    string `json:"doorplate"`
	IsDefault    bool   `json:"isDefault"`
}

type AddressListResponse struct {
	models.Address
	ProvinceName string `json:"provinceName"`
	CityName     string `json:"cityName"`
	DistrictName string `json:"districtName"`
}

// AddressListData 地址列表分页响应数据
type AddressListData struct {
	List  []AddressListResponse `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// GetAddressList 获取当前用户的收货地址列表（分页）
// @Summary 获取地址列表
// @Description 分页获取当前用户的收货地址列表，默认按默认地址优先、创建时间倒序排列
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=AddressListData} "成功返回地址列表"
// @Failure 401 {object} utils.Response "未登录"
// @Router /api/v1/wx/address [get]
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

	var resp []AddressListResponse
	for _, addr := range addresses {
		resp = append(resp, AddressListResponse{
			Address:      addr,
			ProvinceName: utils.GetRegionName(addr.ProvinceID),
			CityName:     utils.GetRegionName(addr.CityID),
			DistrictName: utils.GetRegionName(addr.DistrictID),
		})
	}

	utils.Success(c, AddressListData{
		List:  resp,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// AddAddress 新增地址
// @Summary 新增地址
// @Description 添加一个新的收货地址，如果设置为默认地址，则自动清除其他默认地址
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param request body AddressReq true "地址信息（ID字段可忽略）"
// @Success 200 {object} utils.Response{data=models.Address} "新增成功，返回完整地址对象"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 401 {object} utils.Response "未登录"
// @Router /api/v1/wx/address [post]
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
		ProvinceID:   req.ProvinceID,
		CityID:       req.CityID,
		DistrictID:   req.DistrictID,
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
// @Summary 编辑地址
// @Description 根据ID修改收货地址信息，若设置为默认地址则自动清除其他默认地址
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param id path string true "地址ID"
// @Param request body AddressReq true "地址信息（ID字段会被忽略）"
// @Success 200 {object} utils.Response "更新成功"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 401 {object} utils.Response "未登录"
// @Failure 404 {object} utils.Response "地址不存在"
// @Router /api/v1/wx/address/{id} [put]
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
// @Summary 删除地址
// @Description 根据ID删除指定的收货地址
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param id path string true "地址ID"
// @Success 200 {object} utils.Response "删除成功"
// @Failure 400 {object} utils.Response "无效ID"
// @Failure 401 {object} utils.Response "未登录"
// @Failure 404 {object} utils.Response "地址不存在"
// @Router /api/v1/wx/address/{id} [delete]
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
// @Summary 设置默认地址
// @Description 将指定地址设为默认地址，同时清除其他默认地址
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param id path string true "地址ID"
// @Success 200 {object} utils.Response "设置成功"
// @Failure 400 {object} utils.Response "无效ID"
// @Failure 401 {object} utils.Response "未登录"
// @Failure 404 {object} utils.Response "地址不存在"
// @Router /api/v1/wx/address/{id}/default [put]
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
// @Summary 获取地址详情
// @Description 根据ID获取指定收货地址的详细信息
// @Tags 地址管理
// @Accept json
// @Produce json
// @Param id path string true "地址ID"
// @Success 200 {object} utils.Response{data=models.Address} "成功返回地址详情"
// @Failure 400 {object} utils.Response "无效ID"
// @Failure 401 {object} utils.Response "未登录"
// @Failure 404 {object} utils.Response "地址不存在"
// @Router /api/v1/wx/address/{id} [get]
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
