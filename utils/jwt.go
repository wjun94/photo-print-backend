package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-in-production") // 生产环境从环境变量读取

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	UserType string `json:"user_type"` // "admin" 或 "wx"
	jwt.RegisteredClaims
}

// 生成 token：传入 userID, name, userType
func GenerateToken(userID int64, name string, userType string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: name,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
