package model

import "time"

type TenantNotification struct {
	ID                  uint64 `gorm:"primaryKey"`
	TenantID            uint64 `gorm:"uniqueIndex;not null"`
	Enabled             bool   `gorm:"not null;default:false"`
	WebhookURL          string `gorm:"type:text"`
	Secret              string `gorm:"size:256"`
	PollIntervalMinutes int    `gorm:"not null;default:5"`
	ScenariosJSON       string `gorm:"type:text"`
	ShopIDsJSON         string `gorm:"type:text"`
	AppID               string `gorm:"size:128"`
	AppSecret           string `gorm:"size:256"`
	LastRunAt           *time.Time
	LastRunOK           bool   `gorm:"not null;default:false"`
	LastError           string `gorm:"type:text"`
	LastSentCount       int    `gorm:"not null;default:0"`
	LastBarcodeError    string `gorm:"type:text"`
	NotifiedJSON        string `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (TenantNotification) TableName() string { return "tenant_notifications" }
