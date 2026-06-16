package main

import (
	"log"
	"photo-print-backend/config"
	"photo-print-backend/controllers/admin"
	"photo-print-backend/controllers/app"
	"photo-print-backend/controllers/common"
	"photo-print-backend/database"
	"photo-print-backend/jobs"
	"photo-print-backend/middleware"
	"photo-print-backend/utils"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "photo-print-backend/docs" // swagger docs 生成后引入
)

// @title 照片打印商城 API
// @version 1.0
// @description 网店下单打印照片系统
// @host localhost:8080
// @BasePath /api/v1
func main() {
	config.LoadConfig()
	// 初始化雪花算法，机器ID可从环境变量读取，默认为1
	machineID := config.AppConfig.SnowflakeMachineID
	if machineID == 0 {
		machineID = 1
	}
	utils.InitSnowflake(machineID)
	utils.InitQiniu() // 新增七牛云
	utils.InitWechatPay()
	database.InitDB()
	// 在这里启动你的定时任务
	jobs.StartAutoConfirmJob()
	// 启动订单定时任务 ✅
	jobs.StartAutoCloseOrderJob()
	// 启动删除已取消订单
	jobs.StartDeleteCancelledOrdersJob()
	// 定时清理优惠券
	jobs.InitCronJobs()

	r := gin.Default()
	r.Use(middleware.Cors())

	// 静态文件服务（访问上传的照片）
	r.Static("/uploads", "./uploads")
	// 后台管理页面（静态）
	r.StaticFile("/admin", "./static/admin.html")

	// API 路由组
	api := r.Group("/api/v1")
	{
		// 公开接口
		api.POST("/admin/login", admin.AdminLogin)
		api.POST("/wx/login", app.WxLogin)

		{
			// 微信小程序产品接口
			api.GET("/products", app.GetProductListForWx)
			api.GET("/products/:id", app.GetProductDetailForWx)
			api.GET("/couriers", common.GetCourierList) // 获取快递列表
		}

		// 需要登录的接口（任何有效 token 均可）
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		authorized.POST("/image/delete", common.DeleteImage)
		authorized.GET("/regions/all", common.GetAllRegions)
		{
			// 小程序专用接口（只允许 wx 用户）
			appGroup := authorized.Group("/")
			appGroup.Use(middleware.RequireRole("wx"))
			{
				// 用户与社交关系
				appGroup.GET("/user/info", app.GetUserInfo) // 用户惜
				appGroup.POST("/bind", app.BindInviter)     // 绑定上级接口

				// 媒体/文件上传
				appGroup.POST("/upload/single", common.UploadSingleImage) // 单图（新增）
				appGroup.POST("/upload/batch", common.UploadImages)       // 批量上传(没用到)

				// 工具类
				appGroup.GET("/qrcodes", app.GetQRCodes)

				// 订单业务 (统一使用 /order 根路径)
				appGroup.POST("/order/preview", app.PreviewOrder)
				appGroup.POST("/order/submit", app.SubmitOrder)
				appGroup.GET("/order/:id", app.GetOrderDetail)
				appGroup.GET("/order/list", app.GetWxOrders)        // 我的订单列表
				appGroup.POST("/order/pay/success", app.PaySuccess) // 支付成功
				appGroup.POST("/order/confirm", app.ConfirmReceipt) // 确认收货
				appGroup.POST("/order/cancel", app.CancelOrder)     // 取消订单

				// 微信支付
				appGroup.POST("/pay/order", app.PayOrder)
				appGroup.POST("/pay/notify", app.PayNotify)

				// 收货地址管理 (统一使用 /address)
				appGroup.GET("/address/list", app.GetAddressList)
				appGroup.GET("/address/:id", app.GetAddressDetail)
				appGroup.POST("/address", app.AddAddress)
				appGroup.PUT("/address/:id", app.UpdateAddress)
				appGroup.DELETE("/address/:id", app.DeleteAddress)
				appGroup.PUT("/address/:id/default", app.SetDefaultAddress)

				// 分销/佣金
				appGroup.GET("/commission/total", app.GetTotalCommission)
				appGroup.GET("/commission/list", app.GetCommissionList)
				appGroup.GET("/commission/friends", app.GetInvitedFriends)

				// 优惠券业务
				appGroup.POST("/coupon/receive", app.ReceiveCoupon)
				appGroup.GET("/coupon/list", app.GetMyCoupons)
				appGroup.GET("/coupon/product/:productId", app.GetProductCoupons)
			}

			// 后台管理专用接口（只允许 admin 用户）
			adminGroup := authorized.Group("/admin")
			adminGroup.Use(middleware.RequireRole("admin"))
			{
				// 后台媒体/文件上传
				adminGroup.POST("/upload/single", common.UploadSingleAdminImage) // 单图（新增）
				adminGroup.POST("/upload/batch", common.UploadAdminImages)       // 批量上传(没用到)

				// 二维码管理
				adminGroup.POST("/qrcode/upload", admin.UploadQRCodes)
				adminGroup.GET("/qrcodes", app.GetQRCodes)

				// 数据大屏 / 看板
				adminGroup.GET("/dashboard/overview", admin.GetOverview)
				adminGroup.GET("/dashboard/trend", admin.GetTrend)

				// 管理员信息
				adminGroup.GET("/info", admin.GetAdminInfo)

				// 订单管理
				adminGroup.GET("/orders", admin.GetOrderList)
				adminGroup.GET("/orders/:id", admin.GetOrderDetail)
				adminGroup.PUT("/orders/:id/status", admin.UpdateOrderStatus)
				adminGroup.POST("/orders/ship", admin.ShipOrder)              // 发货
				adminGroup.POST("/orders/complete", admin.AdminCompleteOrder) // 完成订单

				// 微信用户管理
				adminGroup.GET("/wx-users", admin.GetWxUserList)
				adminGroup.PUT("/wx-users/:id/status", admin.SetUserStatus)

				// 商品管理
				adminGroup.POST("/products", admin.CreateProduct)
				adminGroup.PUT("/products/:id", admin.UpdateProduct)
				adminGroup.GET("/products", admin.GetProductList)
				adminGroup.GET("/products/:id", admin.GetProductDetail)
				adminGroup.PUT("/products/:id/status", admin.UpdateProductStatus)
				adminGroup.DELETE("/products/:id", admin.DeleteProduct)

				// 分销分润配置
				adminGroup.POST("/commission/ratio", admin.SetCommissionRatio)
				adminGroup.GET("/commission/ratio", admin.GetCommissionRatio)

				// 优惠券管理营销
				adminGroup.POST("/coupons", admin.CreateCoupon)
				adminGroup.GET("/coupons", admin.GetCouponList)
				adminGroup.GET("/coupons/:id", admin.GetCouponDetail)
				adminGroup.PUT("/coupons/:id", admin.UpdateCoupon)
				adminGroup.DELETE("/coupons/:id", admin.DeleteCoupon)
			}
		}
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Server running on :%s", config.AppConfig.ServerPort)
	r.Run(":" + config.AppConfig.ServerPort)
}
