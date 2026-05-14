package main

import (
	"log"
	"photo-print-backend/config"
	"photo-print-backend/controllers"
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
		api.POST("/admin/login", controllers.AdminLogin)
		api.POST("/wx/login", controllers.WxLogin)

		// 需要登录的接口（任何有效 token 均可）
		authorized := api.Group("/")
		authorized.Use(middleware.AuthMiddleware())
		{
			// 小程序专用接口（只允许 wx 用户）
			wx := authorized.Group("/")
			wx.Use(middleware.RequireRole("wx"))
			{
				wx.POST("/upload", controllers.UploadPhoto)
				wx.POST("/orders", controllers.CreateOrder)
				wx.GET("/orders/:id", controllers.GetOrderDetail)
				wx.GET("/user/info", controllers.GetUserInfo) // 新增
				wx.GET("/orders/wx", controllers.GetWxOrders) // 新增：我的订单列表
			}

			// 后台管理专用接口（只允许 admin 用户）
			admin := authorized.Group("/")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.GET("/orders", controllers.GetOrderList)
				admin.PUT("/orders/:id/status", controllers.UpdateOrderStatus)
				admin.GET("/photos", controllers.GetPhotoList)
			}
		}
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Server running on :%s", config.AppConfig.ServerPort)
	r.Run(":" + config.AppConfig.ServerPort)
}
