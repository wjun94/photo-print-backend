package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"photo-print-backend/config"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

type PayOrderReq struct {
	OrderID string `json:"orderId" binding:"required"`
}

// PayOrder 生成微信支付参数
// @Summary 微信支付下单
// @Description 根据订单ID生成微信预支付订单，返回小程序调起支付所需参数
// @Tags 支付
// @Accept json
// @Produce json
// @Param request body PayOrderReq true "订单ID"
// @Success 200 {object} utils.Response{data=object{nonceStr=string,timeStamp=string,package=string,signType=string,paySign=string}}
// @Router /api/v1/wx/pay [post]
func PayOrder(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		utils.Unauthorized(c, "未登录")
		return
	}

	var req PayOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误")
		return
	}

	orderID, err := strconv.ParseInt(req.OrderID, 10, 64)
	if err != nil {
		utils.Fail(c, "无效订单ID")
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").First(&order, orderID).Error; err != nil {
		utils.Fail(c, "订单不存在")
		return
	}
	if order.UserID.Int64() != userID {
		utils.Fail(c, "订单不属于当前用户")
		return
	}
	if order.Status != models.OrderStatusPending {
		utils.Fail(c, "订单状态不正确，无法支付")
		return
	}

	// 生成商户订单号（使用已有订单号）
	outTradeNo := order.OrderNo
	// 金额单位：分
	totalFee := int64(order.ActualAmount * 100)
	// 商品描述（取第一个商品名称）
	description := "照片打印订单"
	if len(order.Items) > 0 {
		description = order.Items[0].Spec
	}

	// 获取用户 openid
	var wxUser models.WxUser
	if err := database.DB.Select("open_id").Where("id = ?", userID).First(&wxUser).Error; err != nil {
		utils.Fail(c, "获取用户信息失败")
		return
	}

	// 调用微信支付统一下单API
	svc := jsapi.JsapiApiService{Client: utils.WxPayClient}
	resp, result, err := svc.Prepay(c.Request.Context(),
		jsapi.PrepayRequest{
			Appid:       core.String(config.AppConfig.AppId),
			Mchid:       core.String(config.AppConfig.WechatPayMchID),
			Description: core.String(description),
			OutTradeNo:  core.String(outTradeNo),
			NotifyUrl:   core.String(config.AppConfig.WechatPayNotifyURL),
			Amount: &jsapi.Amount{
				Total:    core.Int64(totalFee),
				Currency: core.String("CNY"),
			},
			Attach: core.String(strconv.FormatInt(orderID, 10)), // 附加订单ID，方便回调
			Payer: &jsapi.Payer{
				Openid: core.String(wxUser.OpenID), // 必填
			},
		},
	)
	if err != nil {
		// 打印完整的错误信息
		if result != nil {
			bodyBytes, _ := io.ReadAll(result.Response.Body)
			errMsg := fmt.Sprintf("微信支付下单失败: %s, 状态码: %d, 响应: %s", err.Error(), result.Response.StatusCode, string(bodyBytes))
			log.Println(errMsg)
			utils.Fail(c, errMsg)
		} else {
			log.Println("微信支付下单失败: " + err.Error())
			utils.Fail(c, "支付下单失败: "+err.Error())
		}
		return
	}

	// 获取预支付ID
	prepayID := *resp.PrepayId

	// 生成小程序调起支付的参数
	appId := config.AppConfig.AppId
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceStr := utils.RandomString(32)
	packageStr := "prepay_id=" + prepayID
	signType := "RSA"

	// 计算签名
	signStr := appId + "\n" + timeStamp + "\n" + nonceStr + "\n" + packageStr + "\n"
	paySign, err := utils.SignWithPrivateKey(signStr) // 使用商户私钥签名
	if err != nil {
		utils.Fail(c, "生成签名失败")
		return
	}

	utils.Success(c, gin.H{
		"nonceStr":  nonceStr,
		"timeStamp": timeStamp,
		"package":   packageStr,
		"signType":  signType,
		"paySign":   paySign,
	})
}

// PayNotify 微信支付回调
// @Summary 支付结果通知
// @Tags 支付
// @Accept json
// @Produce json
// @Router /api/v1/pay/notify [post]
func PayNotify(c *gin.Context) {
	var notifyReq struct {
		Resource struct {
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
		} `json:"resource"`
	}
	if err := c.ShouldBindJSON(&notifyReq); err != nil {
		c.AbortWithStatus(400)
		return
	}

	// 解密resource.ciphertext（使用APIv3密钥）
	plaintext, err := utils.DecryptAES256GCM(
		notifyReq.Resource.AssociatedData,
		notifyReq.Resource.Nonce,
		notifyReq.Resource.Ciphertext,
		config.AppConfig.WechatPayApiV3Key,
	)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	var result struct {
		OutTradeNo string `json:"out_trade_no"`
		Attach     string `json:"attach"`
		TradeState string `json:"trade_state"`
	}
	if err := json.Unmarshal([]byte(plaintext), &result); err != nil {
		c.AbortWithStatus(500)
		return
	}

	if result.TradeState == "SUCCESS" {
		orderID, _ := strconv.ParseInt(result.Attach, 10, 64)
		var order models.Order
		if err := database.DB.First(&order, orderID).Error; err == nil {
			if order.Status == models.OrderStatusPending {
				now := utils.LocalTime(time.Now())
				order.Status = models.OrderStatusPaid
				order.PayAt = &now
				database.DB.Save(&order)
			}
		}
	}
	c.String(200, `{"code":"SUCCESS","message":"成功"}`)
}
