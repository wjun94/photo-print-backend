package utils

import (
	"context"
	"photo-print-backend/config"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var WxPayClient *core.Client

func InitWechatPay() {
	// 1. 加载商户私钥（这个保持不变，是你自己的API证书私钥）
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(config.AppConfig.WechatPayPrivateKeyPath)
	if err != nil {
		panic("加载商户API私钥失败: " + err.Error())
	}
	SetMerchantPrivateKey(mchPrivateKey)
	// 2. 获取微信支付公钥ID和公钥内容
	wechatPayPublicKeyId := config.AppConfig.PublicKeySerialNo
	wechatPayPublicKey, err := utils.LoadPublicKeyWithPath(config.AppConfig.PublicKeyPath)
	if err != nil {
		panic("加载微信支付公钥失败: " + err.Error())
	}

	// 3. 使用 WechatPayPublicKeyAuth 创建客户端
	client, err := core.NewClient(
		context.Background(),
		option.WithWechatPayPublicKeyAuthCipher(
			config.AppConfig.WechatPayMchID,
			config.AppConfig.WechatPaySerialNo,
			mchPrivateKey,
			wechatPayPublicKeyId,
			wechatPayPublicKey,
			// config.AppConfig.WechatPayApiV3Key,
		),
	)
	if err != nil {
		panic("初始化微信支付客户端失败: " + err.Error())
	}
	WxPayClient = client
}
