package database

import (
	"fmt"
	"log/slog"

	"github.com/Twelveeee/golib/logger"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func InitDatabase(cfg *config.DatabaseConfig) error {
	// 确保目标数据库存在后建立连接。
	if err := createDatabaseIfNotExists(cfg); err != nil {
		return fmt.Errorf("failed to ensure database: %w", err)
	}

	dsn := buildDSN(cfg, cfg.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.NewGormAdapter(slog.Default()),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	GormDB = db
	return nil
}

func Close() error {
	// 关闭数据库连接。
	if GormDB != nil {
		if sqlDB, err := GormDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func AutoMigrate(models ...interface{}) error {
	// 自动迁移数据库表结构。
	if GormDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return GormDB.AutoMigrate(models...)
}

func createDatabaseIfNotExists(cfg *config.DatabaseConfig) error {
	// 连接 postgres 库检查/创建目标数据库。
	dsn := buildDSN(cfg, "postgres")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.NewGormAdapter(slog.Default()),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	defer sqlDB.Close()

	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", cfg.DBName)
	if err := db.Raw(query).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		createSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
		if err := db.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		slog.Info("Database created", "dbname", cfg.DBName)
	}
	return nil
}

func buildDSN(cfg *config.DatabaseConfig, dbname string) string {
	// 根据配置拼接 DSN。
	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbname, sslmode)
}
