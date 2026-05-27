package common

import (
	"fmt"
	"net/url"
	"path/filepath"
	"photo-print-backend/config"
	"photo-print-backend/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadResponse struct {
	Url string `json:"url"`
}

// UploadImages 上传多张图片到七牛云
// @Summary 上传图片
// @Description 支持单张或多张图片上传（最多9张），存储到七牛云，根据环境选择文件夹
// @Tags 上传
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "图片文件（支持多选）"
// @Success 200 {object} utils.Response{data=[]UploadResponse}
// @Router /api/v1/upload [post]
func UploadImages(c *gin.Context) {
	// 获取当前用户ID（可选，用于记录上传者）
	userID, _ := utils.GetUserID(c)

	form, err := c.MultipartForm()
	if err != nil {
		utils.Fail(c, "解析表单失败")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		utils.Fail(c, "请选择至少一张图片")
		return
	}
	if len(files) > 9 {
		utils.Fail(c, "最多支持9张图片")
		return
	}

	// 确定存储前缀（开发环境 upload-dev/，生产环境 upload/）
	env := config.AppConfig.Env // 需要在 config 中增加 Env 字段
	var prefix string
	if env == "production" {
		prefix = config.AppConfig.UploadPrefixProd
	} else {
		prefix = config.AppConfig.UploadPrefixDev
	}
	// 确保前缀以 / 结尾
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var uploadedURLs []UploadResponse
	for _, file := range files {
		// 文件大小限制 5MB
		if file.Size > 10<<20 {
			utils.Fail(c, fmt.Sprintf("文件 %s 超过10MB限制", file.Filename))
			return
		}
		// 检查扩展名
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			utils.Fail(c, fmt.Sprintf("文件 %s 格式不支持，仅支持 jpg/jpeg/png", file.Filename))
			return
		}

		// 生成七牛云存储路径：前缀 + 时间戳_随机数_用户ID.扩展名
		fileName := fmt.Sprintf("%d_%d_%d%s", time.Now().UnixNano(), userID, time.Now().Unix(), ext)
		key := prefix + fileName

		// 保存临时文件
		tempPath := fmt.Sprintf("/tmp/%s", fileName)
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			utils.Fail(c, fmt.Sprintf("保存文件 %s 失败", file.Filename))
			return
		}

		// 上传到七牛云
		url, err := utils.UploadToQiniu(key, tempPath)
		if err != nil {
			utils.Fail(c, fmt.Sprintf("上传文件 %s 到七牛云失败: %v", file.Filename, err))
			return
		}
		uploadedURLs = append(uploadedURLs, UploadResponse{Url: url})
	}

	utils.Success(c, uploadedURLs)
}

// UploadSingleImage 上传单张图片到七牛云
// @Summary 上传单张图片
// @Description 上传一张图片到七牛云，支持 jpg/jpeg/png，最大5MB
// @Tags 上传
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "图片文件"
// @Success 200 {object} utils.Response{data=UploadResponse}
// @Router /api/v1/upload/single [post]
func UploadSingleImage(c *gin.Context) {
	// 类似 UploadImages 但只处理一个文件
	// 获取用户ID
	userID, _ := utils.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		utils.Fail(c, "请选择图片文件")
		return
	}

	// 验证文件大小和扩展名等
	if file.Size > 10<<20 {
		utils.Fail(c, fmt.Sprintf("文件 %s 超过10MB限制", file.Filename))
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		utils.Fail(c, fmt.Sprintf("文件 %s 格式不支持，仅支持 jpg/jpeg/png", file.Filename))
		return
	}

	// 确定前缀
	env := config.AppConfig.Env
	var prefix string
	if env == "production" {
		prefix = config.AppConfig.UploadPrefixProd
	} else {
		prefix = config.AppConfig.UploadPrefixDev
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// 生成文件名
	fileName := fmt.Sprintf("%d_%d_%d%s", time.Now().UnixNano(), userID, time.Now().Unix(), ext)
	key := prefix + fileName

	// 保存临时文件
	tempPath := fmt.Sprintf("/tmp/%s", fileName)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		utils.Fail(c, fmt.Sprintf("保存文件 %s 失败", file.Filename))
		return
	}

	// 上传到七牛云
	url, err := utils.UploadToQiniu(key, tempPath)
	if err != nil {
		utils.Fail(c, fmt.Sprintf("上传文件 %s 到七牛云失败: %v", file.Filename, err))
		return
	}
	utils.Success(c, UploadResponse{Url: url})
}

// UploadAdminImages 批量上传图片到七牛云（最多9张）
// @Summary 上传图片
// @Description 支持单张或多张图片上传（最多9张），存储到七牛云，根据环境选择文件夹
// @Tags 上传
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "图片文件（支持多选）"
// @Success 200 {object} utils.Response{data=[]UploadResponse}
// @Router /api/v1/upload [post]
func UploadAdminImages(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		utils.Fail(c, "解析表单失败")
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		utils.Fail(c, "未选择任何文件")
		return
	}
	if len(files) > 9 {
		utils.Fail(c, "最多上传9张图片")
		return
	}

	var uploadedURLs []string
	for _, file := range files {
		// 文件校验（大小、扩展名）
		if file.Size > 10<<20 {
			utils.Fail(c, fmt.Sprintf("文件 %s 超过10MB", file.Filename))
			return
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			utils.Fail(c, fmt.Sprintf("文件 %s 格式不支持", file.Filename))
			return
		}

		// 生成key和临时路径
		env := config.AppConfig.Env // 需要在 config 中增加 Env 字段
		var prefix string
		if env == "production" {
			prefix = config.AppConfig.UploadPrefixAdminProd
		} else {
			prefix = config.AppConfig.UploadPrefixAdminDev
		}
		fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		key := prefix + fileName
		tempPath := fmt.Sprintf("/tmp/%s", fileName)
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			utils.Fail(c, fmt.Sprintf("保存文件 %s 失败", file.Filename))
			return
		}

		url, err := utils.UploadToQiniu(key, tempPath)
		if err != nil {
			utils.Fail(c, fmt.Sprintf("上传文件 %s 到七牛云失败", file.Filename))
			return
		}
		uploadedURLs = append(uploadedURLs, url)
	}

	// 返回URL数组
	utils.Success(c, uploadedURLs)
}

// UploadSingleAdminImage 上传单张图片到七牛云
// @Summary 上传单张图片
// @Description 上传一张图片到七牛云，支持 jpg/jpeg/png，最大5MB
// @Tags 上传
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "图片文件"
// @Success 200 {object} utils.Response{data=UploadResponse}
// @Router /api/v1/upload/single [post]
func UploadSingleAdminImage(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		utils.Fail(c, "请选择图片文件")
		return
	}

	// 限制 10MB
	if file.Size > 10<<20 {
		utils.Fail(c, "文件不能超过10MB")
		return
	}

	// 校验扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		utils.Fail(c, "只支持 jpg、jpeg、png 格式")
		return
	}

	env := config.AppConfig.Env // 需要在 config 中增加 Env 字段
	var prefix string
	if env == "production" {
		prefix = config.AppConfig.UploadPrefixAdminProd
	} else {
		prefix = config.AppConfig.UploadPrefixAdminDev
	}
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	key := prefix + fileName

	// 保存到临时文件
	tempPath := fmt.Sprintf("/tmp/%s", fileName)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		utils.Fail(c, "保存文件失败")
		return
	}

	// 上传到七牛云
	url, err := utils.UploadToQiniu(key, tempPath)
	if err != nil {
		utils.Fail(c, fmt.Sprintf("上传失败: %v", err))
		return
	}

	utils.Success(c, gin.H{
		"url": url,
	})
}

type DeleteImageReq struct {
	Url string `json:"url" binding:"required"`
}

// DeleteImage 删除七牛云图片
func DeleteImage(c *gin.Context) {
	var req DeleteImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, "参数错误：缺少 imageUrl")
		return
	}

	// 解析 URL 获取 key
	parsedURL, err := url.Parse(req.Url)
	if err != nil {
		utils.Fail(c, "无效的图片URL")
		return
	}
	// 去掉域名部分，得到路径如 "upload/2026/05/23/xxx.jpg"
	domain := config.AppConfig.QiniuDomain
	key := strings.TrimPrefix(parsedURL.Path, "/")
	if key == parsedURL.Path {
		// 如果没有去掉斜杠，尝试去掉域名
		key = strings.TrimPrefix(req.Url, domain)
		key = strings.TrimPrefix(key, "/")
	}
	if key == "" {
		utils.Fail(c, "无法解析图片key")
		return
	}

	// 调用七牛云删除
	if err := utils.DeleteFromQiniu(key); err != nil {
		utils.Fail(c, "删除图片失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message": "删除成功",
	})
}
