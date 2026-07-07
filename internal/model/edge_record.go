package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	EdgeRecordTypePacking   = "packing"
	EdgeRecordTypeUnboxing  = "unboxing"
	EdgeRecordStatusDraft     = "draft"
	EdgeRecordStatusRecording = "recording"
	EdgeRecordStatusCompleted = "completed"
	CloudEdgeID = "cloud"
)

type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported photo_paths type %T", value)
	}
}

type EdgeRecord struct {
	ID             int64       `gorm:"primaryKey" json:"id"`
	EdgeID         string      `gorm:"column:edge_id;not null;default:''" json:"edgeId"`
	Type           string      `gorm:"not null" json:"type"`
	TrackingNumber string      `gorm:"column:tracking_number;not null" json:"trackingNumber"`
	Status         string      `gorm:"not null;default:draft" json:"status"`
	VideoPath      string      `gorm:"column:video_path" json:"videoPath"`
	PhotoPaths     StringSlice `gorm:"column:photo_paths;type:jsonb" json:"photoPaths"`
	Remark         string      `json:"remark"`
	CreatedAt      time.Time   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time   `gorm:"column:updated_at" json:"updatedAt"`
	CompletedAt    *time.Time  `gorm:"column:completed_at" json:"completedAt"`
}

func (EdgeRecord) TableName() string { return "after_sales.edge_record" }
