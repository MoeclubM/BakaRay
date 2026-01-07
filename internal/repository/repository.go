package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bakaray/internal/config"
	"bakaray/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库连接
var DB *gorm.DB

// NewDB 创建数据库连接
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Type) {
	case "mysql", "mariadb":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
		dialector = mysql.Open(dsn)
	default:
		dbPath := cfg.Path
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join("/app", dbPath)
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		dialector = sqlite.Open(dbPath)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	DB = db
	return db, nil
}

// NewRedis 创建 Redis 连接（可选）
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	// 如果 Redis Host 为空，跳过 Redis 初始化
	if cfg.Host == "" {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) error {
	// 先执行自动迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserGroup{},
		&models.Node{},
		&models.NodeAllowedGroup{},
		&models.NodeGroup{},
		&models.ForwardingRule{},
		&models.Target{},
		&models.GostRule{},
		&models.IPTablesRule{},
		&models.Package{},
		&models.Order{},
		&models.PaymentConfig{},
		&models.PaymentProvider{},
		&models.SiteConfig{},
		&models.TrafficLog{},
	); err != nil {
		return err
	}

	// SQLite 手动添加新列（GORM AutoMigrate 对 SQLite 支持有限）
	return addMissingColumns(db)
}

// addMissingColumns 确保数据库有新添加的列
func addMissingColumns(db *gorm.DB) error {
	// 为 Package 表添加 visible 和 renewable 列
	columns := []struct {
		Name    string
		Type    string
		Default string
	}{
		{"visible", "BOOLEAN", "1"},
		{"renewable", "BOOLEAN", "0"},
	}

	// 检测数据库类型
	isMySQL := db.Config.Dialector.Name() == "mysql"

	for _, col := range columns {
		var exists bool
		var err error

		if isMySQL {
			// MySQL: 使用 information_schema 检查列是否存在
			var count int64
			query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'packages' AND column_name = '%s'", col.Name)
			err = db.Raw(query).Scan(&count).Error
			exists = count > 0
		} else {
			// SQLite: 使用 PRAGMA table_info
			var count int64
			query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('packages') WHERE name='%s'", col.Name)
			err = db.Raw(query).Scan(&count).Error
			exists = count > 0
		}

		if err != nil {
			continue
		}

		if !exists {
			if isMySQL {
				// MySQL: 使用 ALTER TABLE ADD COLUMN
				addQuery := fmt.Sprintf("ALTER TABLE packages ADD COLUMN %s %s DEFAULT %s", col.Name, col.Type, col.Default)
				if err := db.Exec(addQuery).Error; err != nil {
					// 忽略错误（列可能已存在）
					continue
				}
			} else {
				// SQLite: 使用 ALTER TABLE ADD COLUMN
				addQuery := fmt.Sprintf("ALTER TABLE packages ADD COLUMN %s %s DEFAULT %s", col.Name, col.Type, col.Default)
				if err := db.Exec(addQuery).Error; err != nil {
					// 忽略错误（列可能已存在）
					continue
				}
			}
		}
	}

	return nil
}
