package model

import "time"

const (
	EdgeDeviceStatusOnline  = "online"
	EdgeDeviceStatusOffline = "offline"
	EdgeDeviceStatusUnknown = "unknown"
)

type EdgeDevice struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	EdgeID     string     `gorm:"size:64;uniqueIndex;not null" json:"edgeId"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	BaseURL    string     `gorm:"size:512" json:"baseUrl"`
	Status     string     `gorm:"size:32;not null;default:unknown" json:"status"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
	Remark     string     `gorm:"type:text" json:"remark"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (EdgeDevice) TableName() string { return "edge_devices" }
