package model

import "time"

const (
	UnboxingStatusDraft     = "draft"
	UnboxingStatusCompleted = "completed"
)

type UnboxingRecord struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	TenantID         uint64    `gorm:"index;not null" json:"tenantId"`
	TrackingNo       string    `gorm:"size:64;not null" json:"trackingNo"`
	VideoObjectKey   string    `gorm:"size:512" json:"videoObjectKey"`
	VideoURL         string    `gorm:"size:1024" json:"videoUrl"`
	VideoSize        int64     `json:"videoSize"`
	VideoDurationSec int       `json:"videoDurationSec"`
	VideoMimeType    string    `gorm:"size:64" json:"videoMimeType"`
	Status           string    `gorm:"size:32;not null;default:draft" json:"status"`
	Remark           string    `gorm:"type:text" json:"remark"`
	OperatorID       uint64    `json:"operatorId"`
	OperatorName     string    `gorm:"size:64" json:"operatorName"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Photos           []UnboxingPhoto `gorm:"foreignKey:RecordID" json:"photos,omitempty"`
}

func (UnboxingRecord) TableName() string { return "unboxing_records" }

type UnboxingPhoto struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"index;not null" json:"tenantId"`
	RecordID    uint64    `gorm:"index;not null" json:"recordId"`
	ObjectKey   string    `gorm:"size:512" json:"objectKey"`
	PhotoURL    string    `gorm:"size:1024" json:"photoUrl"`
	IssueRemark string    `gorm:"type:text" json:"issueRemark"`
	SortOrder   int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (UnboxingPhoto) TableName() string { return "unboxing_photos" }
