package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{Code: 1, Message: msg})
}

func Error(c *gin.Context, msg string, code int) {
	c.JSON(code, Response{Code: code, Message: msg})
}
