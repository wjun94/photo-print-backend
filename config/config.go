package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	ServerPort         string
	SnowflakeMachineID int64
	Env                string // development / production

	// 运费
	FixedFreight       float64 // 固定运费
	FreeShippingAmount float64 // 包邮门槛,满额包邮（0表示不启用）

	// 新增七牛云配置
	QiniuAccessKey string
	QiniuSecretKey string
	QiniuBucket    string
	QiniuDomain    string // 存储空间绑定的域名（如 http://cdn.example.com）
	QiniuZone      string // 区域：z0（华东）、z1（华北）、z2（华南）、na0（北美）、as0（东南亚）

	UploadPrefixDev       string // 开发环境上传目录前缀，如 "upload-dev/"
	UploadPrefixProd      string // 生产环境上传目录前缀，如 "upload/"
	UploadPrefixAdminDev  string // 开发环境上传目录前缀，如 "upload-admin-dev/"
	UploadPrefixAdminProd string // 生产环境上传目录前缀，如 "upload-admin/"

	AppId     string
	AppSecret string
}

var AppConfig *Config

func LoadConfig() {
	AppConfig = &Config{
		DBHost:     getEnv("DB_HOST", "db"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "123456"),
		DBName:     getEnv("DB_NAME", "photoprint"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
	AppConfig.SnowflakeMachineID = getEnvInt64("SNOWFLAKE_MACHINE_ID", 1)
	AppConfig.Env = getEnv("ENV", "development")

	// 运费
	AppConfig.FixedFreight = getEnvFloat64("FIXED_FREIGHT", 10.0)
	AppConfig.FreeShippingAmount = getEnvFloat64("FREE_SHIPPING_AMOUNT", 28.0)

	// 新增七牛云配置
	AppConfig.QiniuAccessKey = getEnv("QINIU_ACCESS_KEY", "")
	AppConfig.QiniuSecretKey = getEnv("QINIU_SECRET_KEY", "")
	AppConfig.QiniuBucket = getEnv("QINIU_BUCKET", "photo-print")
	AppConfig.QiniuDomain = getEnv("QINIU_DOMAIN", "http://your-domain.com")
	AppConfig.QiniuZone = getEnv("QINIU_ZONE", "z0")
	AppConfig.UploadPrefixDev = getEnv("UPLOAD_PREFIX_DEV", "upload-dev/")
	AppConfig.UploadPrefixProd = getEnv("UPLOAD_PREFIX_PROD", "upload/")
	AppConfig.UploadPrefixAdminDev = getEnv("UPLOAD_PREFIX_ADMIN_DEV", "upload-admin-dev/")
	AppConfig.UploadPrefixAdminProd = getEnv("UPLOAD_PREFIX_ADMIN_PROD", "upload-admin/")
	AppConfig.AppId = getEnv("APPID", "")
	AppConfig.AppSecret = getEnv("APPSECRET", "")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}
