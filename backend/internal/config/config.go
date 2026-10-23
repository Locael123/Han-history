// Package config 汇总后端的全部可调参数。
//
// 所有参数都可以用环境变量覆盖，默认值面向「单机 SQLite + 小内存服务器」。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 是后端的全部配置。
type Config struct {
	AppEnv    string // dev / prod
	Addr      string // 监听地址
	DataDir   string // 数据目录
	DBPath    string // SQLite 文件
	StaticDir string // 前端产物目录，留空则只提供 API

	JWTSecret     string
	JWTTTL        time.Duration
	AllowRegister bool

	DefaultPageSize int
	MaxPageSize     int

	// SQLite 连接与性能参数
	MaxOpenConns int
	MaxIdleConns int
	CacheSizeKB  int
	BusyTimeout  time.Duration
	MMapSize     int64
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Printf("[config] %s=%q 不是合法整数，使用默认值 %d", key, v, def)
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDur(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// Load 读取环境变量并构造配置。
func Load() *Config {
	appEnv := env("APP_ENV", "dev")
	dataDir := env("DATA_DIR", "./data")

	c := &Config{
		AppEnv:          appEnv,
		Addr:            env("ADDR", ":8090"),
		DataDir:         dataDir,
		DBPath:          env("DB_PATH", filepath.Join(dataDir, "gonghe.db")),
		StaticDir:       env("STATIC_DIR", "../web/dist"),
		JWTSecret:       env("JWT_SECRET", ""),
		JWTTTL:          envDur("JWT_TTL", 7*24*time.Hour),
		AllowRegister:   envBool("ALLOW_REGISTER", true),
		DefaultPageSize: envInt("DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:     envInt("MAX_PAGE_SIZE", 100),
		MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 4),
		MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 4),
		CacheSizeKB:     envInt("DB_CACHE_KB", 16*1024),
		BusyTimeout:     envDur("DB_BUSY_TIMEOUT", 5*time.Second),
		MMapSize:        int64(envInt("DB_MMAP_MB", 32)) * 1024 * 1024,
	}

	if c.JWTSecret == "" {
		if c.IsProd() {
			log.Println("[config] 警告: 生产环境未设置 JWT_SECRET，已生成临时密钥（重启后所有登录失效）")
		}
		c.JWTSecret = randomSecret()
	}
	return c
}

func randomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// DSN 返回 SQLite 连接串（pure-Go 驱动 modernc.org/sqlite 的参数风格）。
func (c *Config) DSN() string {
	busy := int(c.BusyTimeout.Milliseconds())
	if busy <= 0 {
		busy = 5000
	}
	pragmas := []string{
		"_pragma=busy_timeout(" + strconv.Itoa(busy) + ")",
		"_pragma=journal_mode(WAL)",
		"_pragma=synchronous(NORMAL)",
		"_pragma=temp_store(2)",
		"_pragma=cache_size(-" + strconv.Itoa(c.CacheSizeKB) + ")",
		"_pragma=foreign_keys(1)",
		"_pragma=mmap_size(" + strconv.FormatInt(c.MMapSize, 10) + ")",
		"_time_format=sqlite",
		"_txlock=immediate",
	}
	return "file:" + filepath.ToSlash(c.DBPath) + "?" + strings.Join(pragmas, "&")
}

// IsProd 用于决定日志级别等。
func (c *Config) IsProd() bool { return c.AppEnv == "prod" }
