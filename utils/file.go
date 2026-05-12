package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// SaveUploadedFile 保存上传的文件并返回文件路径
func SaveUploadedFile(file *multipart.FileHeader, uploadDir string) (string, error) {
	// 确保目录存在
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", err
	}
	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	return filename, nil
}
