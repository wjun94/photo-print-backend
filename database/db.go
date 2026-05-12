package database

import (
	"fmt"
	"log"
	"photo-print-backend/config"
	"photo-print-backend/models"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	cfg := config.AppConfig
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("连接数据库失败: ", err)
	}
	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移表
	err = DB.AutoMigrate(&models.Photo{}, &models.Order{}, &models.OrderItem{}, &models.User{})
	if err != nil {
		log.Fatal("迁移失败: ", err)
	}
	log.Println("数据库连接成功，表已准备就绪")
	// 创建默认管理员（如果不存在）
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		// hashed, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{Username: "admin", Password: "admin123"}
		DB.Create(&admin)
		log.Println("默认管理员已创建: admin / admin123")
	}
}
