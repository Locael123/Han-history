// Package response 统一 API 返回结构。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 是所有接口的统一信封：code=0 表示成功。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	CodeOK           = 0
	CodeBadRequest   = 40000
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeTooManyReq   = 42900
	CodeServerError  = 50000
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeOK, Message: "ok", Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: CodeOK, Message: "created", Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Message: msg})
}

func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, CodeBadRequest, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = "请先登录"
	}
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Forbidden(c *gin.Context, msg string) {
	Fail(c, http.StatusForbidden, CodeForbidden, msg)
}

func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "资源不存在"
	}
	Fail(c, http.StatusNotFound, CodeNotFound, msg)
}

func Conflict(c *gin.Context, msg string) {
	Fail(c, http.StatusConflict, CodeConflict, msg)
}

func TooMany(c *gin.Context, msg string) {
	Fail(c, http.StatusTooManyRequests, CodeTooManyReq, msg)
}

func ServerError(c *gin.Context, msg string) {
	if msg == "" {
		msg = "服务器内部错误"
	}
	Fail(c, http.StatusInternalServerError, CodeServerError, msg)
}

// Page 分页结果。
type Page[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func NewPage[T any](list []T, total int64, page, pageSize int) Page[T] {
	if list == nil {
		list = []T{}
	}
	return Page[T]{List: list, Total: total, Page: page, PageSize: pageSize}
}
