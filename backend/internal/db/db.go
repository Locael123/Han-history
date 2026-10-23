// Package db 负责打开 SQLite、迁移表结构与建立索引。
package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gonghe/internal/config"
	"gonghe/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 打开数据库并完成迁移。
func Open(cfg *config.Config) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	newLogger := logger.New(
		log.New(os.Stdout, "[gorm] ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             300 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	gdb, err := gorm.Open(sqlite.Open(cfg.DSN()), &gorm.Config{
		Logger:         newLogger,
		PrepareStmt:    true,
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := gdb.AutoMigrate(
		&model.User{},
		&model.Branch{},
		&model.Event{},
		&model.Like{},
		&model.Comment{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	if err := createIndexes(gdb); err != nil {
		return nil, fmt.Errorf("创建索引失败: %w", err)
	}
	return gdb, nil
}

// createIndexes 用原生 SQL 建索引：GORM tag 无法表达排序方向与复合列顺序，
// 而列表页最常用的恰恰是「按热度倒序」「按年份正序」这类查询。
func createIndexes(gdb *gorm.DB) error {
	stmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_events_branch_seq ON events(branch_id, id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_timeline ON events(year, month, day, id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_hot ON events(like_count DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_branches_parent ON branches(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_branches_official_hot ON branches(official, like_count DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_branches_owner ON branches(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_target ON comments(target_type, target_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_likes_user ON likes(user_id, target_type)`,
	}
	for _, stmt := range stmts {
		if err := gdb.Exec(stmt).Error; err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}

// Close 优雅关闭数据库连接。
func Close(gdb *gorm.DB) {
	if gdb == nil {
		return
	}
	if sqlDB, err := gdb.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
