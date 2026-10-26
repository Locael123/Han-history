// 中华共和实录·野 —— 服务入口。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gonghe/internal/config"
	"gonghe/internal/db"
	"gonghe/internal/handler"
	"gonghe/internal/router"
	"gonghe/internal/service"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("[boot] 打开数据库失败: %v", err)
	}
	defer db.Close(gdb)

	if err := service.Seed(gdb, cfg.SeedDemo); err != nil {
		log.Fatalf("[boot] 初始化史稿失败: %v", err)
	}

	branches := service.NewBranchService(gdb, cfg)
	events := service.NewEventService(gdb, cfg, branches)
	interactions := service.NewInteractionService(gdb, cfg)
	h := handler.New(cfg, gdb, branches, events, interactions)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router.New(cfg, h),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[boot] 中华共和实录·野 已开卷，监听 %s（环境 %s）", cfg.Addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("[boot] 正在合卷……")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[boot] 关闭超时: %v", err)
	}
}
