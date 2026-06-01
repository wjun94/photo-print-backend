package admin

import (
	"photo-print-backend/database" // 数据库连接与操作封装
	"photo-print-backend/models"   // 数据模型定义（订单、物流等）
	"photo-print-backend/utils"    // 通用工具函数（ID生成、时间处理、响应封装）
	"strconv"                      // 字符串与数字转换
	"time"                         // 时间处理

	"github.com/gin-gonic/gin" // Gin Web框架
	"gorm.io/gorm"             // GORM ORM框架
)

// ShipOrderReq 管理员发货请求参数结构体
// 支持分批发货场景，每次调用创建一条独立的物流记录
type ShipOrderReq struct {
	OrderID     string `json:"orderId" binding:"required"`     // 订单ID（字符串类型，前端传递）
	TrackingNo  string `json:"trackingNo" binding:"required"`  // 快递单号
	CourierCode string `json:"courierCode" binding:"required"` // 快递公司编码（用于对接物流查询接口）
	CourierName string `json:"courierName" binding:"required"` // 快递公司名称（用于前端展示）
	Remark      string `json:"remark"`                         // 可选备注
}

// ShipOrder 管理员发货接口
// 业务特性：
// 1. 支持分批发货：同一个订单可以多次调用此接口，每次生成一条物流记录
// 2. 状态流转控制：仅允许已支付/已发货状态的订单添加物流
// 3. 事务保证：物流记录创建与订单状态更新原子性执行
// 4. 时间统一：所有时间使用本地时区时间，避免跨时区问题
// @Summary 管理员发货
// @Tags 订单管理
// @Accept json
// @Produce json
// @Param request body ShipOrderReq true "发货请求参数"
// @Success 200 {object} utils.Response "发货成功"
// @Failure 400 {object} utils.Response "参数错误/订单不存在/状态不允许"
// @Failure 500 {object} utils.Response "发货失败（数据库错误）"
// @Router /admin/orders/ship [post]
func ShipOrder(c *gin.Context) {
	// 1. 绑定并校验请求参数
	var req ShipOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	// 2. 转换订单ID类型：前端传递字符串，数据库存储int64
	orderID, err := strconv.ParseInt(req.OrderID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效订单ID")
		return
	}

	// 3. 查询订单基本信息
	var order models.Order
	if err := database.DB.First(&order, orderID).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}

	// 4. 订单状态合法性校验
	// 允许状态：
	// - OrderStatusPaid：已支付未发货（首次发货场景）
	// - OrderStatusShipped：已发货（分批发货场景，允许追加包裹）
	// 禁止状态：待支付、已取消、已完成、已退款
	if order.Status != models.OrderStatusPaid && order.Status != models.OrderStatusShipped {
		utils.Fail(c, "只有已支付或已发货的订单才能发货")
		return
	}

	// 5. 初始化物流记录
	logistics := models.Logistics{
		ID:          utils.Int64Str(utils.NextID()), // 生成分布式唯一ID并转为字符串
		OrderID:     order.ID,                       // 关联订单ID
		TrackingNo:  req.TrackingNo,                 // 快递单号
		CourierCode: req.CourierCode,                // 快递公司编码
		CourierName: req.CourierName,                // 快递公司名称
		Remark:      req.Remark,
		CreatedAt:   utils.LocalTime(time.Now()), // 创建时间（本地时区）
		UpdatedAt:   utils.LocalTime(time.Now()), // 更新时间（本地时区）
	}

	// 6. 判断是否需要更新订单主状态
	// 仅当订单当前为"已支付"状态时，才更新为"已发货"并记录首次发货时间
	// 分批发货时（已发货状态），不修改订单主状态和发货时间
	now := utils.LocalTime(time.Now())
	needUpdateOrder := false
	if order.Status == models.OrderStatusPaid {
		order.Status = models.OrderStatusShipped // 更新订单状态为已发货
		order.ShippedAt = &now                   // 记录首次发货时间
		needUpdateOrder = true
	}

	// 7. 数据库事务：保证物流记录创建与订单状态更新的原子性
	// 任意一步失败都会回滚所有操作，避免数据不一致
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 7.1 创建物流记录
		if err := tx.Create(&logistics).Error; err != nil {
			return err
		}

		// 7.2 如果是首次发货，更新订单主信息
		if needUpdateOrder {
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
		}

		// 事务提交
		return nil
	})

	// 8. 事务执行结果处理
	if err != nil {
		utils.Fail(c, "发货失败")
		return
	}

	// 9. 返回成功响应
	utils.Success(c, nil)
}
