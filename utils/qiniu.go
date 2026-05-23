package utils

import (
	"context"
	"fmt"
	"photo-print-backend/config"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

var (
	mac       *qbox.Mac
	bucket    string
	domain    string
	uploadCfg *storage.Config
)

// InitQiniu 初始化七牛云
func InitQiniu() {
	cfg := config.AppConfig
	if cfg.QiniuAccessKey == "" || cfg.QiniuSecretKey == "" {
		fmt.Errorf("七牛云未配置，上传功能不可用")
		return
	}
	mac = qbox.NewMac(cfg.QiniuAccessKey, cfg.QiniuSecretKey)
	bucket = cfg.QiniuBucket
	domain = cfg.QiniuDomain

	// 根据配置的区域选择 zone
	var zone *storage.Zone
	switch cfg.QiniuZone {
	case "z0":
		zone = &storage.ZoneHuadong
	case "z1":
		zone = &storage.ZoneHuabei
	case "z2":
		zone = &storage.ZoneHuanan
	case "na0":
		zone = &storage.ZoneBeimei
	case "as0":
		zone = &storage.ZoneXinjiapo
	default:
		zone = &storage.ZoneHuadong
	}

	uploadCfg = &storage.Config{
		Zone:          zone,
		UseHTTPS:      true,
		UseCdnDomains: false,
	}
}

// UploadToQiniu 上传文件到七牛云
// key: 存储的文件名（可带路径前缀）
// localFile: 本地文件路径
// 返回文件的公开 URL
func UploadToQiniu(key string, localFile string) (string, error) {
	if mac == nil {
		return "", fmt.Errorf("七牛云未初始化")
	}
	putPolicy := storage.PutPolicy{
		Scope: bucket + ":" + key,
	}
	upToken := putPolicy.UploadToken(mac)

	formUploader := storage.NewFormUploader(uploadCfg)
	ret := storage.PutRet{}

	err := formUploader.PutFile(context.Background(), &ret, upToken, key, localFile, nil)
	if err != nil {
		return "", err
	}
	// 返回完整 URL
	// return fmt.Sprintf("%s/%s", domain, key), nil
	return key, nil
}

// DeleteFromQiniu 从七牛云删除文件
func DeleteFromQiniu(key string) error {
	if mac == nil {
		return fmt.Errorf("七牛云未初始化")
	}
	bucketManager := storage.NewBucketManager(mac, uploadCfg)
	err := bucketManager.Delete(bucket, key)
	if err != nil {
		return err
	}
	return nil
}
