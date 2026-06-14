package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

var merchantPrivateKey *rsa.PrivateKey

// SetMerchantPrivateKey 设置商户私钥（在初始化支付时调用）
func SetMerchantPrivateKey(key *rsa.PrivateKey) {
	merchantPrivateKey = key
}

// RandomString 生成随机字符串
func RandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

// SignWithPrivateKey 使用商户私钥进行 SHA256 with RSA 签名
func SignWithPrivateKey(data string) (string, error) {
	if merchantPrivateKey == nil {
		return "", fmt.Errorf("商户私钥未初始化")
	}
	hashed := sha256.Sum256([]byte(data))
	signature, err := rsa.SignPKCS1v15(rand.Reader, merchantPrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}
