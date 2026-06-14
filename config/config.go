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

	UploadPrefix       string // 开发环境上传目录前缀，如 "upload-dev/"
	UploadPrefixAdmin  string // 开发环境上传目录前缀，如 "upload-admin-dev/"
	UploadPrefixQrcode string // 开发环境上传目录前缀，如 "upload-admin/"

	// 小程序配置
	AppId     string
	AppSecret string
	// 微信支付配置
	WechatPayMchID          string // 商户号
	WechatPayApiV3Key       string // APIv3密钥
	WechatPaySerialNo       string // 商户证书序列号
	WechatPayPrivateKeyPath string // 商户私钥路径
	WechatPayNotifyURL      string // 支付结果回调地址
	PublicKeyPath           string // 商户使用微信支付公钥验签
	PublicKeySerialNo       string // 公钥序列号
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
	AppConfig.FreeShippingAmount = getEnvFloat64("FREE_SHIPPING_AMOUNT", 10.0)

	// 新增七牛云配置
	AppConfig.QiniuAccessKey = getEnv("QINIU_ACCESS_KEY", "")
	AppConfig.QiniuSecretKey = getEnv("QINIU_SECRET_KEY", "")
	AppConfig.QiniuBucket = getEnv("QINIU_BUCKET", "photo-print")
	AppConfig.QiniuDomain = getEnv("QINIU_DOMAIN", "http://your-domain.com")
	AppConfig.QiniuZone = getEnv("QINIU_ZONE", "z0")
	AppConfig.UploadPrefix = getEnv("UPLOAD_PREFIX", "")
	AppConfig.UploadPrefixAdmin = getEnv("UPLOAD_PREFIX_ADMIN", "")
	AppConfig.UploadPrefixQrcode = getEnv("UPLOAD_PREFIX_QRCODE", "")
	AppConfig.AppId = getEnv("APPID", "")
	AppConfig.AppSecret = getEnv("APPSECRET", "")

	// 微信支付
	AppConfig.WechatPayMchID = getEnv("WECHATPAYMCHID", "")
	AppConfig.WechatPayApiV3Key = getEnv("WECHATAPIV3", "")
	AppConfig.WechatPayPrivateKeyPath = getEnv("WECHATPAYPRIVATEKEYPATH", "")
	AppConfig.WechatPaySerialNo = getEnv("WECHATPAYPRIVATECERT", "")
	AppConfig.PublicKeyPath = getEnv("PUBLICKEYPATH", "")
	AppConfig.PublicKeySerialNo = getEnv("PUBLICKEYSERIALNO", "")
	AppConfig.WechatPayNotifyURL = getEnv("WECHATPAYNOTIFYURL", "")
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
