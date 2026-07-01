package database

import (
	"fmt"
	"os"
	"path/filepath"

	"aftersalescore/internal/config"
	"aftersalescore/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.UnboxingRecord{}, &model.UnboxingPhoto{}); err != nil {
		return err
	}
	if db.Dialector.Name() == "postgres" {
		return db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_unboxing_tenant_tracking ON unboxing_records (tenant_id, tracking_no);
			CREATE INDEX IF NOT EXISTS idx_unboxing_created ON unboxing_records (tenant_id, created_at DESC);
		`).Error
	}
	return nil
}
