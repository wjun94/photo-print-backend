package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret JWT 签名密钥
// 注意：生产环境**严禁硬编码**，建议从环境变量、配置中心读取
var jwtSecret = []byte("your-secret-key-change-in-production")

// Claims 自定义 JWT 载荷结构体
// 包含用户基础信息 + JWT 标准注册声明
type Claims struct {
	// 自定义字段：用户ID、用户名、用户类型
	UserID   int64  `json:"user_id"`   // 用户唯一ID
	Username string `json:"username"`  // 用户名
	UserType string `json:"user_type"` // 用户类型：admin=管理员, wx=微信用户

	// JWT 标准注册声明（内置过期时间、签发时间等）
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
// 参数：
//   userID   - 用户ID
//   name     - 用户名
//   userType - 用户类型(admin/wx)
// 返回：
//   签名后的 token 字符串、可能出现的错误
func GenerateToken(userID int64, name string, userType string) (string, error) {
	// 构造自定义载荷信息
	claims := Claims{
		UserID:   userID,
		Username: name,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 30 * time.Hour)), // 过期时间：24 * 30小时
			IssuedAt:  jwt.NewNumericDate(time.Now()),                          // 签发时间：当前时间
		},
	}

	// 使用 HS256 算法签名，创建 token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥进行签名，返回最终 token 字符串
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并验证 JWT Token
// 参数：tokenString - 前端传递的 token 字符串
// 返回：
//   解析后的 Claims 结构体指针、验证过程中的错误（过期、无效、签名错误等）
func ParseToken(tokenString string) (*Claims, error) {
	// 解析 token，同时验证签名和合法性
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 返回密钥用于验证签名
		return jwtSecret, nil
	})

	// 解析过程出错（格式错误、签名错误、过期等）
	if err != nil {
		return nil, err
	}

	// 类型断言，将解析后的 Claims 转为自定义结构体，并验证 token 是否有效
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	// 无效 token
	return nil, jwt.ErrTokenInvalidClaims
}
