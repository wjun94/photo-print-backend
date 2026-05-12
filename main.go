package main

import (
	"log"
	"photo-print-backend/config"
	"photo-print-backend/controllers"
	"photo-print-backend/database"
	"photo-print-backend/middleware"

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
		api.POST("/login", controllers.Login)
		api.POST("/upload", controllers.UploadPhoto)       // 小文件上传可以公开或单独保护
		api.POST("/orders", controllers.CreateOrder)       // 下单也可公开（需用户ID）
		api.GET("/orders/:id", controllers.GetOrderDetail) // 查询订单公开（后续可加签名）

		// 需要登录的后台接口
		authApi := api.Group("/")
		authApi.Use(middleware.AuthMiddleware())
		{
			authApi.GET("/orders", controllers.GetOrderList) // 后台订单列表
			authApi.PUT("/orders/:id/status", controllers.UpdateOrderStatus)
			authApi.GET("/photos", controllers.GetPhotoList)
		}
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Server running on :%s", config.AppConfig.ServerPort)
	r.Run(":" + config.AppConfig.ServerPort)
}
