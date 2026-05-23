package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetWxUserList 获取微信用户列表
// @Summary 微信用户列表
// @Description 后台管理微信用户，支持昵称/手机号搜索、禁用状态筛选、创建时间区间查询
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param keyword query string false "搜索关键词（昵称/手机号）"
// @Param isDisabled query bool false "是否禁用（true/false）"
// @Param createdAtStart query string false "创建时间起始，格式 2006-01-02 15:04:05"
// @Param createdAtEnd query string false "创建时间结束，格式 2006-01-02 15:04:05"
// @Success 200 {object} utils.Response{data=object{list=[]models.WxUser,total=int64,page=int,size=int}}
// @Router /api/v1/admin/wx-users [get]
func GetWxUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	statusStr := c.Query("status") // 可传 "0" 或 "1"
	createdAtStart := c.Query("createdAtStart")
	createdAtEnd := c.Query("createdAtEnd")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	offset := (page - 1) * size

	var users []models.WxUser
	var total int64

	query := database.DB.Model(&models.WxUser{})

	// 关键词搜索（昵称或手机号）
	if keyword != "" {
		query = query.Where("nickname LIKE ? OR mobile LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 禁用状态筛选
	if statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		query = query.Where("status = ?", status)
	}

	// 创建时间区间
	if createdAtStart != "" {
		query = query.Where("created_at >= ?", createdAtStart)
	}
	if createdAtEnd != "" {
		query = query.Where("created_at <= ?", createdAtEnd)
	}

	// 计数
	query.Count(&total)

	// 分页查询
	query.Offset(offset).Limit(size).Order("created_at desc").Find(&users)

	utils.Success(c, gin.H{
		"list":  users,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// SetUserDisabled 设置用户禁用/启用状态
// @Summary 设置用户禁用状态
// @Description 管理员操作：禁用或启用微信用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param disabled body object true "状态" example(isDisabled=true)
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/wx-users/{id}/disable [put]
func SetUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Fail(c, "无效ID")
		return
	}

	var req struct {
		Status int `json:"status" binding:"required,oneof=0 1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误，status 必须为 0 或 1")
		return
	}

	result := database.DB.Model(&models.WxUser{}).Where("id = ?", id).Update("status", req.Status)
	if result.RowsAffected == 0 {
		utils.Fail(c, "用户不存在")
		return
	}
	utils.Success(c, nil)
}
