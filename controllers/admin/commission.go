package admin

import (
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type SetRatioReq struct {
	Ratio float64 `json:"ratio" binding:"required,min=0,max=100" example:"15"` // 返利百分比
}

// SetCommissionRatio 设置返利比例
// @Summary 设置返利比例
// @Description 管理员调整推广返利百分比，会新增一条配置记录
// @Tags 佣金管理
// @Accept json
// @Produce json
// @Param body body SetRatioReq true "返利比例"
// @Success 200 {object} utils.Response
// @Router /api/v1/admin/commission/ratio [post]
func SetCommissionRatio(c *gin.Context) {
	var req SetRatioReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误: "+err.Error())
		return
	}
	adminID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}
	setting := models.CommissionSetting{
		Ratio:     req.Ratio,
		UpdatedBy: strconv.FormatInt(adminID, 10),
		UpdatedAt: utils.LocalTime(time.Now()),
	}
	if err := database.DB.Create(&setting).Error; err != nil {
		utils.Fail(c, "设置失败")
		return
	}
	utils.Success(c, nil)
}

// GetCommissionRatio 获取当前生效的返利比例
// @Summary 获取返利比例
// @Description 获取最新设置的返利百分比
// @Tags 佣金管理
// @Produce json
// @Success 200 {object} utils.Response{data=object{ratio=float64}}
// @Router /api/v1/admin/commission/ratio [get]
func GetCommissionRatio(c *gin.Context) {
	var setting models.CommissionSetting
	err := database.DB.Order("updated_at desc").First(&setting).Error
	if err != nil {
		utils.Success(c, gin.H{"ratio": 10.0})
		return
	}
	utils.Success(c, gin.H{"ratio": setting.Ratio})
}
