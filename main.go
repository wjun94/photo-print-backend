package main

import (
	"log"
	"photo-print-backend/config"
	"photo-print-backend/controllers/admin"
	"photo-print-backend/controllers/app"
	"photo-print-backend/controllers/common"
	"photo-print-backend/database"
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
	database.InitDB()

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

		// 需要登录的接口（任何有效 token 均可）
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		authorized.POST("/image/delete", common.DeleteImage)
		{
			// 小程序专用接口（只允许 wx 用户）
			appGroup := authorized.Group("/")
			appGroup.Use(middleware.RequireRole("wx"))
			{
				appGroup.POST("/upload/single", common.UploadSingleImage) // 单图（新增）
				appGroup.POST("/upload/batch", common.UploadImages)       // 批量上传(没用到)

				appGroup.POST("/orders", app.CreateOrder)
				appGroup.GET("/orders/:id", app.GetOrderDetail)
				appGroup.GET("/user/info", app.GetUserInfo) // 新增
				appGroup.GET("/orders/wx", app.GetWxOrders) // 新增：我的订单列表

				appGroup.GET("/products", app.GetProductListForWx)
				appGroup.GET("/products/:id", app.GetProductDetailForWx)
			}

			// 后台管理专用接口（只允许 admin 用户）
			adminGroup := authorized.Group("/admin")
			adminGroup.Use(middleware.RequireRole("admin"))
			{
				adminGroup.POST("/upload/single", common.UploadSingleAdminImage) // 单图（新增）
				adminGroup.POST("/upload/batch", common.UploadAdminImages)       // 批量上传(没用到)

				adminGroup.GET("/orders", admin.GetOrderList)
				adminGroup.GET("/info", admin.GetAdminInfo)
				adminGroup.PUT("/orders/:id/status", admin.UpdateOrderStatus)

				adminGroup.GET("/wx-users", admin.GetWxUserList)
				adminGroup.PUT("/wx-users/:id/status", admin.SetUserStatus)

				// 商品管理
				adminGroup.POST("/products", admin.CreateProduct)
				adminGroup.GET("/products", admin.GetProductList)
				adminGroup.GET("/products/:id", admin.GetProductDetail)
				adminGroup.PUT("/products/:id", admin.UpdateProduct)
				adminGroup.PUT("/products/:id/status", admin.UpdateProductStatus)
				adminGroup.DELETE("/products/:id", admin.DeleteProduct)
			}
		}
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Server running on :%s", config.AppConfig.ServerPort)
	r.Run(":" + config.AppConfig.ServerPort)
}
