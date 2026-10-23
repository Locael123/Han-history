// Package handler 把 HTTP 请求翻译成 service 调用。
package handler

import (
	"errors"
	"log"
	"strconv"

	"gonghe/internal/config"
	"gonghe/internal/middleware"
	"gonghe/internal/response"
	"gonghe/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 持有全部依赖。构造一次，路由里按方法取用。
type Handler struct {
	Cfg          *config.Config
	DB           *gorm.DB
	Branches     *service.BranchService
	Events       *service.EventService
	Interactions *service.InteractionService
}

func New(cfg *config.Config, db *gorm.DB, branches *service.BranchService, events *service.EventService, interactions *service.InteractionService) *Handler {
	return &Handler{Cfg: cfg, DB: db, Branches: branches, Events: events, Interactions: interactions}
}

// fail 把服务层错误映射成统一信封。非 AppError 一律视为 500，
// 只把原始错误写进日志，不把内部细节漏给客户端。
func fail(c *gin.Context, err error) {
	var appErr *service.AppError
	if errors.As(err, &appErr) {
		response.Fail(c, appErr.Status, codeOf(appErr.Status), appErr.Message)
		return
	}
	log.Printf("[api] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	response.ServerError(c, "")
}

func codeOf(status int) int {
	switch status {
	case 400:
		return response.CodeBadRequest
	case 401:
		return response.CodeUnauthorized
	case 403:
		return response.CodeForbidden
	case 404:
		return response.CodeNotFound
	case 409:
		return response.CodeConflict
	case 429:
		return response.CodeTooManyReq
	default:
		return response.CodeServerError
	}
}

func bind(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.BadRequest(c, "请求内容格式有误")
		return false
	}
	return true
}

// pathID 解析 :id。非法值按 404 处理，避免把内部错误暴露出去。
func pathID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		response.NotFound(c, "资源不存在")
		return 0, false
	}
	return id, true
}

// queryInt 读取正整数查询参数，非法时退回默认值。
func queryInt(c *gin.Context, key string, def, max int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func (h *Handler) page(c *gin.Context) (int, int) {
	return queryInt(c, "page", 1, 0), queryInt(c, "pageSize", h.Cfg.DefaultPageSize, h.Cfg.MaxPageSize)
}

func userID(c *gin.Context) uint64 { return middleware.UserID(c) }
