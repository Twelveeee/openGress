package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Twelveeee/golib/logger"
	"github.com/Twelveeee/openGress/backend/pkg/app"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/database"
)

func main() {
	// 加载配置与日志，初始化数据库与应用生命周期。
	cfg, err := config.LoadConfig("config/config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 用于优雅退出的根上下文。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化结构化日志。
	logg, closeFunc, err := logger.NewLogger(ctx, &cfg.LogConfig)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := closeFunc(); err != nil {
			log.Printf("logger shutdown error: %v", err)
		}
	}()
	if logg != nil {
		slog.SetDefault(logg)
	}

	slog.InfoContext(ctx, "Configuration loaded successfully")

	// 初始化数据库与表结构。
	if err := database.InitDatabase(&cfg.Database); err != nil {
		panic(err)
	}
	if err := database.AutoMigrate(
		&database.PlayerModel{},
		&database.PlayerInventoryModel{},
		&database.PortalModel{},
		&database.LinkModel{},
		&database.FieldModel{},
		&database.LogModel{},
		&database.AdminBanModel{},
		&database.AuthUserModel{},
	); err != nil {
		panic(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.ErrorContext(ctx, "Database close failed", "err", err)
		}
	}()

	// 启动 HTTP/WS 服务。
	application := app.New(cfg)
	application.Start(ctx)
	defer func() {
		if err := application.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "Shutdown failed", "err", err)
		}
	}()

	// 监听退出信号并触发优雅关闭。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.InfoContext(ctx, "Shutting down gracefully...")
	cancel()
}
