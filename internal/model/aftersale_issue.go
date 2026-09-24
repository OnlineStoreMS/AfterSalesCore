package model

import "time"

// AftersaleIssue 售后问题记录（人工处理类问题，区别于平台固有售后状态）。
type AftersaleIssue struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	TenantID  uint64 `gorm:"index;not null" json:"tenantId"`
	ShopID    uint64 `gorm:"index" json:"shopId"`
	ShopName  string `gorm:"size:128" json:"shopName"`

	OrderNo             string `gorm:"size:64;index" json:"orderNo"`
	PlatformOrderID     string `gorm:"size:128;index" json:"platformOrderId"`
	PlatformAftersaleID string `gorm:"size:64;index" json:"platformAftersaleId"`

	ProductTitle string `gorm:"size:512" json:"productTitle"`
	ProductImage string `gorm:"size:2048" json:"productImage"`
	SkuSpecs     string `gorm:"size:512" json:"skuSpecs"`

	BuyerName    string `gorm:"size:128" json:"buyerName"`
	BuyerPhone   string `gorm:"size:32" json:"buyerPhone"`
	BuyerAddress string `gorm:"size:1024" json:"buyerAddress"`

	HasAftersale      bool   `gorm:"default:false" json:"hasAftersale"`
	AftersaleStatus   string `gorm:"size:64" json:"aftersaleStatus"`
	AftersaleType     string `gorm:"size:64" json:"aftersaleType"`
	Logistics         string `gorm:"size:512" json:"logistics"`
	ReturnLogisticsNo string `gorm:"size:64" json:"returnLogisticsNo"`
	ShipLogisticsNo   string `gorm:"size:64" json:"shipLogisticsNo"`

	ProblemType       string `gorm:"size:64;index;not null" json:"problemType"`
	ProblemTypeCustom string `gorm:"size:128" json:"problemTypeCustom"`
	HandleMethod      string `gorm:"size:64" json:"handleMethod"`
	HandleMethodCustom string `gorm:"size:128" json:"handleMethodCustom"`
	ProblemNote       string `gorm:"type:text" json:"problemNote"`
	HandleNote        string `gorm:"type:text" json:"handleNote"`

	Status string `gorm:"size:32;index;not null;default:pending" json:"status"`

	OperatorID   uint64 `json:"operatorId"`
	OperatorName string `gorm:"size:64" json:"operatorName"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CreatedAt   time.Time  `gorm:"index" json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (AftersaleIssue) TableName() string { return "aftersale_issues" }

const (
	IssueStatusPending    = "pending"
	IssueStatusProcessing = "processing"
	IssueStatusDone       = "done"

	IssueProblemCustom = "custom"
	IssueHandleCustom  = "custom"
)
