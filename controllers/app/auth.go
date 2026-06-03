package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"photo-print-backend/database"
	"photo-print-backend/models"
	"photo-print-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// WxLoginReq 小程序登录请求参数
type WxLoginReq struct {
	Code       string `json:"code" binding:"required"`
	InviteCode string `json:"inviteCode"` // 可选，邀请人的用户ID（上级）
}

// WxLogin 小程序静默登录（支持绑定邀请码）
// @Summary 小程序静默登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param login body WxLoginReq true "登录请求（含inviteCode）"
// @Success 200 {object} utils.Response{data=object{token=string,userId=string}}
// @Router /api/v1/wx/login [post]
func WxLogin(c *gin.Context) {
	var req WxLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数绑定失败: "+err.Error())
		return
	}

	// 1. 配置微信参数 (建议从配置文件读取，这里写死仅作演示)
	appID := "wx7ce22bfac91039a9"                   // 替换为你的实际 AppID
	appSecret := "dd79e79b9d02b6e7665f0427a7d43bea" // 替换为你的实际 AppSecret

	// 2. 构建微信接口请求 URL
	// 注意：实际生产环境请使用配置管理，不要硬编码
	wxURL := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(appID),
		url.QueryEscape(appSecret),
		url.QueryEscape(req.Code))

	// 3. 发起 HTTP GET 请求
	resp, err := http.Get(wxURL)
	if err != nil {
		utils.Fail(c, "微信服务请求超时")
		return
	}
	defer resp.Body.Close()

	// 4. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Fail(c, "读取微信响应失败")
		return
	}

	// 5. 解析微信返回的 JSON
	var wxResp struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid,omitempty"` // 如果绑定了开放平台会有此字段
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &wxResp); err != nil {
		utils.Fail(c, "解析微信数据失败")
		return
	}

	// 6. 处理微信返回的错误码
	if wxResp.ErrCode != 0 {
		// 常见错误码处理
		switch wxResp.ErrCode {
		case 40029:
			utils.Fail(c, "登录凭证(code)无效")
		case 45011:
			utils.Fail(c, "频率限制，每个用户每分钟最多100次")
		default:
			utils.Fail(c, "微信登录失败: "+wxResp.ErrMsg)
		}
		return
	}

	// 7. 核心业务逻辑：使用真实的 OpenID 查找或创建用户
	var user models.WxUser
	result := database.DB.Where("open_id = ?", wxResp.OpenID).First(&user)
	isNewUser := result.Error != nil

	if isNewUser {
		// 新用户注册逻辑
		user = models.WxUser{OpenID: wxResp.OpenID}
		// ... (后续的邀请码处理逻辑保持不变)
		if req.InviteCode != "" {
			inviterID, err := strconv.ParseInt(req.InviteCode, 10, 64)
			if err == nil && inviterID != 0 {
				var inviter models.WxUser
				if database.DB.First(&inviter, inviterID).Error == nil && inviter.Status == 1 {
					user.InviterID = utils.Int64Str(inviterID)
				}
			}
		}
		database.DB.Create(&user)
	} else {
		// 老用户直接登录
		// 如果需要更新最后登录时间，可以在这里 Update
	}

	// 8. 生成 Token 返回
	token, err := utils.GenerateToken(user.ID.Int64(), "", "wx")
	if err != nil {
		utils.Fail(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{
		"token":  token,
		"userId": user.ID.String(),
		"isNew":  isNewUser, // 可选：告诉前端是否是新用户，用于引导完善资料
	})
}
