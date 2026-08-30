package model

import "time"

const (
	ShopPlatformDoudian   = "doudian"
	ShopPlatformTaobao    = "taobao"
	ShopPlatformPinduoduo = "pinduoduo"

	ShopPluginUnbound = "unbound"
	ShopPluginBound   = "bound"
	ShopPluginOnline  = "online"
	ShopPluginOffline = "offline"
)

type MarketplaceShop struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	TenantID         uint64     `gorm:"index;not null" json:"tenantId"`
	Name             string     `gorm:"size:128;not null" json:"name"`
	Platform         string     `gorm:"size:32;not null;index" json:"platform"`
	BindCode         string     `gorm:"size:16;uniqueIndex;not null" json:"bindCode"`
	PluginKey        string     `gorm:"size:64;index" json:"pluginKey"`
	PluginSecretHash string     `gorm:"size:128" json:"-"`
	PluginStatus     string     `gorm:"size:32;not null;default:unbound" json:"pluginStatus"`
	PlatformShopID   string     `gorm:"size:64" json:"platformShopId"`
	PlatformShopName string     `gorm:"size:128" json:"platformShopName"`
	LastSyncAt       *time.Time `json:"lastSyncAt"`
	LastSeenAt       *time.Time `json:"lastSeenAt"`
	SyncRequestedAt  *time.Time `json:"syncRequestedAt"`
	Remark           string     `gorm:"type:text" json:"remark"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (MarketplaceShop) TableName() string { return "marketplace_shops" }

type AftersaleFilterCard struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TenantID  uint64    `gorm:"index;not null" json:"tenantId"`
	ShopID    uint64    `gorm:"uniqueIndex:uk_shop_card;index;not null" json:"shopId"`
	GroupName string    `gorm:"size:64;not null" json:"groupName"`
	CardKey   string    `gorm:"size:128;uniqueIndex:uk_shop_card;not null" json:"cardKey"`
	CardLabel string    `gorm:"size:64;not null" json:"cardLabel"`
	Count     int       `gorm:"not null;default:0" json:"count"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	SyncedAt  time.Time `json:"syncedAt"`
}

func (AftersaleFilterCard) TableName() string { return "aftersale_filter_cards" }

type AftersaleTicket struct {
	ID                  uint64                `gorm:"primaryKey" json:"id"`
	TenantID            uint64                `gorm:"index;not null" json:"tenantId"`
	ShopID              uint64                `gorm:"uniqueIndex:uk_shop_aftersale;index;not null" json:"shopId"`
	PlatformAftersaleID string                `gorm:"size:64;uniqueIndex:uk_shop_aftersale;not null" json:"platformAftersaleId"`
	OrderNo             string                `gorm:"size:64;index" json:"orderNo"`
	ProductTitle        string                `gorm:"size:512" json:"productTitle"`
	ProductImage        string                `gorm:"size:2048" json:"productImage"`
	SKU                 string                `gorm:"size:256" json:"sku"`
	ProductTags         string                `gorm:"size:256" json:"productTags"`
	Tags                string                `gorm:"size:256" json:"tags"`
	Qty                 int                   `json:"qty"`
	BuyQty              int                   `json:"buyQty"`
	PayAmount           string                `gorm:"size:32" json:"payAmount"`
	RefundAmount        string                `gorm:"size:32" json:"refundAmount"`
	AftersaleType       string                `gorm:"size:64" json:"aftersaleType"`
	Reason              string                `gorm:"size:256" json:"reason"`
	Status              string                `gorm:"size:64;index" json:"status"`
	TimeoutText         string                `gorm:"size:256" json:"timeoutText"`
	TimeoutAction       string                `gorm:"size:128" json:"timeoutAction"`
	DeadlineAt          *time.Time            `gorm:"index" json:"deadlineAt"`
	Dispute             string                `gorm:"size:128" json:"dispute"`
	Logistics           string                `gorm:"size:256" json:"logistics"`
	ReturnLogisticsNo   string                `gorm:"size:64;index" json:"returnLogisticsNo"`
	ApplyTime           string                `gorm:"size:64" json:"applyTime"`
	RawJSON             string                `gorm:"type:text" json:"rawJson"`
	SyncedAt            time.Time             `json:"syncedAt"`
	CreatedAt           time.Time             `json:"createdAt"`
	UpdatedAt           time.Time             `json:"updatedAt"`
	CardKeys            []AftersaleTicketCard `gorm:"foreignKey:TicketID" json:"cardKeys,omitempty"`
}

func (AftersaleTicket) TableName() string { return "aftersale_tickets" }

type AftersaleTicketCard struct {
	ID       uint64 `gorm:"primaryKey" json:"id"`
	TicketID uint64 `gorm:"uniqueIndex:uk_ticket_card;index;not null" json:"ticketId"`
	CardKey  string `gorm:"size:128;uniqueIndex:uk_ticket_card;not null" json:"cardKey"`
}

func (AftersaleTicketCard) TableName() string { return "aftersale_ticket_cards" }

type ServiceOrder struct {
	ID                uint64     `gorm:"primaryKey" json:"id"`
	TenantID          uint64     `gorm:"index;not null" json:"tenantId"`
	ShopID            uint64     `gorm:"uniqueIndex:uk_shop_service;index;not null" json:"shopId"`
	PlatformServiceID string     `gorm:"size:64;uniqueIndex:uk_shop_service;not null" json:"platformServiceId"`
	OrderNo           string     `gorm:"size:64;index" json:"orderNo"`
	ProductTitle      string     `gorm:"size:512" json:"productTitle"`
	ProductImage      string     `gorm:"size:2048" json:"productImage"`
	ProductContent    string     `gorm:"size:256" json:"productContent"`
	BuyerNick         string     `gorm:"size:128" json:"buyerNick"`
	CreateSource      string     `gorm:"size:64" json:"createSource"`
	BusinessType      string     `gorm:"size:64;index" json:"businessType"`
	OrderType         string     `gorm:"size:64" json:"orderType"`
	Tags              string     `gorm:"size:256" json:"tags"`
	StatusTab         string     `gorm:"size:32;index" json:"statusTab"`
	Status            string     `gorm:"size:64;index" json:"status"`
	TimeoutText       string     `gorm:"size:256" json:"timeoutText"`
	TimeoutAction     string     `gorm:"size:128" json:"timeoutAction"`
	DeadlineAt        *time.Time `gorm:"index" json:"deadlineAt"`
	DelayEndTime      int64      `json:"delayEndTime"`
	Detail            string     `gorm:"type:text" json:"detail"`
	Solution          string     `gorm:"size:256" json:"solution"`
	LastLog           string     `gorm:"size:256" json:"lastLog"`
	LastLogTime       string     `gorm:"size:64" json:"lastLogTime"`
	CreateTime        string     `gorm:"size:64" json:"createTime"`
	RawJSON           string     `gorm:"type:text" json:"rawJson"`
	SyncedAt          time.Time  `json:"syncedAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func (ServiceOrder) TableName() string { return "service_orders" }

type ReturnPackage struct {
	ID                  uint64    `gorm:"primaryKey" json:"id"`
	TenantID            uint64    `gorm:"index;not null" json:"tenantId"`
	ShopID              uint64    `gorm:"uniqueIndex:uk_shop_return;index;not null" json:"shopId"`
	PlatformAftersaleID string    `gorm:"size:64;uniqueIndex:uk_shop_return;not null" json:"platformAftersaleId"`
	OrderNo             string    `gorm:"size:64;index" json:"orderNo"`
	ProductTitle        string    `gorm:"size:512" json:"productTitle"`
	ProductImage        string    `gorm:"size:2048" json:"productImage"`
	SKU                 string    `gorm:"size:256" json:"sku"`
	Qty                 int       `json:"qty"`
	PayAmount           string    `gorm:"size:32" json:"payAmount"`
	RefundAmount        string    `gorm:"size:32" json:"refundAmount"`
	AftersaleType       string    `gorm:"size:64" json:"aftersaleType"`
	Reason              string    `gorm:"size:256" json:"reason"`
	Status              string    `gorm:"size:64" json:"status"`
	Logistics           string    `gorm:"size:256" json:"logistics"`
	LogisticsNo         string    `gorm:"size:64;index" json:"logisticsNo"`
	Carrier             string    `gorm:"size:64" json:"carrier"`
	ReturnLocation      string     `gorm:"size:512;index" json:"returnLocation"`
	ShipTime            string     `gorm:"size:64" json:"shipTime"`
	ApplyTime           string     `gorm:"size:64" json:"applyTime"`
	ReturnTime          string     `gorm:"size:64" json:"returnTime"`
	ReturnedAt          *time.Time `gorm:"index" json:"returnedAt"`
	AppliedAt           *time.Time `gorm:"index" json:"appliedAt"`
	TrackJSON           string     `gorm:"type:text" json:"trackJson"`
	RawJSON             string    `gorm:"type:text" json:"rawJson"`
	SyncedAt            time.Time `json:"syncedAt"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func (ReturnPackage) TableName() string { return "return_packages" }
