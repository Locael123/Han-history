// Package router 装配所有路由。
package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gonghe/internal/config"
	"gonghe/internal/handler"
	"gonghe/internal/middleware"
	"gonghe/internal/response"

	"github.com/gin-gonic/gin"
)

// New 构造 Gin 引擎。API 在 /api/v1 下，其余路径交给前端产物。
func New(cfg *config.Config, h *handler.Handler) *gin.Engine {
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Logger(), middleware.Recovery(), middleware.BodyLimit())
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	api := r.Group("/api/v1")
	api.Use(middleware.OptionalAuth(cfg))

	// 登录 / 注册：独立限流，防止撞库。
	authLimiter := middleware.NewRateLimiter(0.5, 8)
	api.POST("/auth/register", authLimiter.Middleware(), h.Register)
	api.POST("/auth/login", authLimiter.Middleware(), h.Login)
	api.GET("/auth/me", middleware.JWTAuth(cfg), h.Me)

	// 公开读接口。OptionalAuth 让登录用户能看到自己的点赞状态。
	api.GET("/stats", h.Stats)
	api.GET("/graph", h.Graph)
	api.GET("/branches", h.ListBranches)
	api.GET("/branches/:id", h.DetailBranch)
	api.GET("/branches/:id/timeline", h.BranchTimeline)
	api.GET("/events", h.ListEvents)
	api.GET("/events/:id", h.DetailEvent)
	api.GET("/comments", h.Comments)

	// 写接口一律要求登录。
	write := api.Group("")
	write.Use(middleware.JWTAuth(cfg))
	{
		write.POST("/branches", h.CreateBranch)
		write.PATCH("/branches/:id", h.UpdateBranch)
		write.DELETE("/branches/:id", h.DeleteBranch)
		write.POST("/branches/:id/events", h.CreateEvent)
		write.PATCH("/events/:id", h.UpdateEvent)
		write.DELETE("/events/:id", h.DeleteEvent)
		write.POST("/likes", h.ToggleLike)
		write.POST("/comments", h.CreateComment)
		write.DELETE("/comments/:id", h.DeleteComment)
		write.GET("/me/branches", h.MyBranches)
		write.GET("/me/events", h.MyEvents)
	}

	serveStatic(r, cfg.StaticDir)
	return r
}

// serveStatic 托管前端构建产物；找不到文件时回落到 index.html，
// 让前端路由自己处理深链接。
func serveStatic(r *gin.Engine, dir string) {
	if dir == "" {
		r.NoRoute(func(c *gin.Context) { response.NotFound(c, "接口不存在") })
		return
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		r.NoRoute(func(c *gin.Context) { response.NotFound(c, "接口不存在") })
		return
	}
	index := filepath.Join(root, "index.html")
	if st, err := os.Stat(index); err != nil || st.IsDir() {
		r.NoRoute(func(c *gin.Context) { response.NotFound(c, "接口不存在") })
		return
	}
	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		if strings.HasPrefix(urlPath, "/api/") {
			response.NotFound(c, "接口不存在")
			return
		}
		clean := filepath.Join(root, filepath.Clean("/"+urlPath))
		// filepath.Join 已经把 .. 清理掉，但再做一次前缀检查，双重保险。
		if !strings.HasPrefix(clean, root+string(os.PathSeparator)) && clean != root {
			c.File(index)
			return
		}
		if st, err := os.Stat(clean); err == nil && !st.IsDir() {
			c.File(clean)
			return
		}
		c.File(index)
	})
}
