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
		api.POST("/upload", controllers.UploadPhoto)
		api.POST("/orders", controllers.CreateOrder)
		api.GET("/orders", controllers.GetOrderList)
		api.GET("/orders/:id", controllers.GetOrderDetail)
		api.PUT("/orders/:id/status", controllers.UpdateOrderStatus)
		api.GET("/photos", controllers.GetPhotoList)
	}

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Server running on :%s", config.AppConfig.ServerPort)
	r.Run(":" + config.AppConfig.ServerPort)
}
