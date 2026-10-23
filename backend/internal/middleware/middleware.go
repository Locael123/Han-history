// Package middleware 提供鉴权、限流、日志等 Gin 中间件。
package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gonghe/internal/auth"
	"gonghe/internal/config"
	"gonghe/internal/response"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes 是请求体上限。条目正文上限一万字，1MB 足够，
// 同时防止「一个超大 JSON 打满内存」。
const MaxBodyBytes = 1 << 20

// BodyLimit 限制请求体大小。
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)
		}
		c.Next()
	}
}

const (
	ctxUserID   = "currentUserID"
	ctxUsername = "currentUsername"
)

// JWTAuth 强制校验登录态。
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			raw = c.Query("token")
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if tokenStr == "" {
			response.Unauthorized(c, "请先登录")
			return
		}
		claims, err := auth.Parse(cfg, tokenStr)
		if err != nil {
			response.Unauthorized(c, err.Error())
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxUsername, claims.Username)
		c.Next()
	}
}

// OptionalAuth 在令牌存在时解析用户，不存在时按游客继续。
// 用于公开的浏览接口：游客能看，登录用户还能看到「我是否赞过」。
func OptionalAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		tokenStr := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if tokenStr != "" {
			if claims, err := auth.Parse(cfg, tokenStr); err == nil {
				c.Set(ctxUserID, claims.UserID)
				c.Set(ctxUsername, claims.Username)
			}
		}
		c.Next()
	}
}

// UserID 从上下文取出当前用户 ID，游客为 0。
func UserID(c *gin.Context) uint64 {
	v, ok := c.Get(ctxUserID)
	if !ok {
		return 0
	}
	id, _ := v.(uint64)
	return id
}

// Username 从上下文取出当前用户名。
func Username(c *gin.Context) string {
	v, ok := c.Get(ctxUsername)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Logger 打印访问日志（只记方法、路径、状态、耗时）。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" && c.Request.Method == http.MethodGet {
			path += "?" + raw
		}
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		if status >= 500 {
			log.Printf("[http] %s %s %d %s", c.Request.Method, path, status, latency.Round(time.Millisecond))
		}
	}
}

// Recovery 兜住 panic，返回统一信封而不是空响应。
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Printf("[panic] %s %s: %v", c.Request.Method, c.Request.URL.Path, recovered)
		response.ServerError(c, "服务器内部错误")
	})
}

// ---------------- 令牌桶限流 ----------------

type bucket struct {
	tokens   float64
	lastSeen int64
}

// RateLimiter 按 IP 限流，用于登录、注册等敏感接口。
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	capacity float64
	lastGC   int64
}

// NewRateLimiter 创建限流器：perSecond 是平均速率，burst 是突发容量。
func NewRateLimiter(perSecond, burst float64) *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*bucket), rate: perSecond, capacity: burst}
}

func (l *RateLimiter) allow(key string, now int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, lastSeen: now}
		l.buckets[key] = b
	}
	elapsed := float64(now-b.lastSeen) / 1e9
	b.tokens = min(l.capacity, b.tokens+elapsed*l.rate)
	b.lastSeen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *RateLimiter) needGC(now int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lastGC == 0 {
		l.lastGC = now
		return false
	}
	return now-l.lastGC > int64(5*time.Minute)
}

func (l *RateLimiter) gc(now int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, b := range l.buckets {
		if now-b.lastSeen > int64(10*time.Minute) {
			delete(l.buckets, k)
		}
	}
	l.lastGC = now
}

// Middleware 返回 Gin 中间件。
func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().UnixNano()
		if !l.allow(c.ClientIP(), now) {
			response.TooMany(c, "操作过于频繁，请稍后再试")
			return
		}
		if l.needGC(now) {
			go l.gc(now)
		}
		c.Next()
	}
}
