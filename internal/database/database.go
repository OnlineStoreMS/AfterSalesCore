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
	if err := db.AutoMigrate(
		&model.UnboxingRecord{},
		&model.UnboxingPhoto{},
		&model.EdgeDevice{},
		&model.MarketplaceShop{},
		&model.TenantSetting{},
		&model.AftersaleFilterCard{},
		&model.AftersaleTicket{},
		&model.AftersaleTicketCard{},
		&model.ServiceOrder{},
		&model.ReturnPackage{},
		&model.TenantNotification{},
	); err != nil {
		return err
	}
	if db.Dialector.Name() == "postgres" {
		if err := db.Exec(`
			CREATE SCHEMA IF NOT EXISTS after_sales;
			CREATE TABLE IF NOT EXISTS after_sales.edge_record (
				id BIGSERIAL PRIMARY KEY,
				edge_id TEXT NOT NULL DEFAULT '',
				type TEXT NOT NULL CHECK (type IN ('packing', 'unboxing')),
				tracking_number TEXT NOT NULL,
				status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'recording', 'completed')),
				video_path TEXT NOT NULL DEFAULT '',
				photo_paths JSONB NOT NULL DEFAULT '[]'::jsonb,
				remark TEXT NOT NULL DEFAULT '',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				completed_at TIMESTAMPTZ
			);
			CREATE INDEX IF NOT EXISTS idx_edge_record_tracking ON after_sales.edge_record (tracking_number);
			CREATE INDEX IF NOT EXISTS idx_edge_record_type ON after_sales.edge_record (type);
			CREATE INDEX IF NOT EXISTS idx_edge_record_edge ON after_sales.edge_record (edge_id);
			CREATE INDEX IF NOT EXISTS idx_edge_record_created ON after_sales.edge_record (created_at DESC);
		`).Error; err != nil {
			return err
		}
		return db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_unboxing_tenant_tracking ON unboxing_records (tenant_id, tracking_no);
			CREATE INDEX IF NOT EXISTS idx_unboxing_created ON unboxing_records (tenant_id, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_edge_devices_edge_id ON edge_devices (edge_id);
			UPDATE return_packages
			SET applied_at = to_timestamp(apply_time, 'YYYY/MM/DD HH24:MI:SS')
			WHERE applied_at IS NULL
			  AND apply_time ~ '^[0-9]{4}/[0-9]{1,2}/[0-9]{1,2}';
		`).Error
	}
	return nil
}
