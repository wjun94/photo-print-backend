package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// DecryptAES256GCM 解密微信支付回调的 AEAD_AES_256_GCM 加密数据
func DecryptAES256GCM(associatedData, nonce, ciphertext, apiV3Key string) (string, error) {
	// 解码密文
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %v", err)
	}
	// 解码 nonce
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("解码nonce失败: %v", err)
	}
	// 密钥必须为32字节
	keyBytes := []byte(apiV3Key)
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("APIv3密钥长度必须为32字节")
	}
	// 创建 AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %v", err)
	}
	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %v", err)
	}
	// 解密
	plaintext, err := gcm.Open(nil, nonceBytes, ciphertextBytes, []byte(associatedData))
	if err != nil {
		return "", fmt.Errorf("解密失败: %v", err)
	}
	return string(plaintext), nil
}
