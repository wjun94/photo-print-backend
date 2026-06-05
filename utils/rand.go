package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// 生成纯数字随机昵称，指定长度
func RandNumberNickName(length int) string {
	if length <= 0 {
		length = 6
	}
	var sb strings.Builder
	// 数字字符集
	chars := "0123456789"
	max := big.NewInt(int64(len(chars)))

	for i := 0; i < length; i++ {
		// 安全随机
		n, _ := rand.Int(rand.Reader, max)
		sb.WriteByte(chars[n.Int64()])
	}
	return sb.String()
}

// 生成 字母+数字 随机昵称（更美观）
func RandMixNickName(length int) string {
	if length <= 0 {
		length = 8
	}
	var sb strings.Builder
	// 字母+数字
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	max := big.NewInt(int64(len(chars)))

	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, max)
		sb.WriteByte(chars[n.Int64()])
	}
	return sb.String()
}

// 生成带前缀的昵称，如 user_1234
func RandPrefixNickName(prefix string, numLen int) string {
	return prefix + "_" + RandNumberNickName(numLen)
}
