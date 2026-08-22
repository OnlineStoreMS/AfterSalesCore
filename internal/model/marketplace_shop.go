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
	ProductImage        string                `gorm:"size:1024" json:"productImage"`
	SKU                 string                `gorm:"size:256" json:"sku"`
	Qty                 int                   `json:"qty"`
	PayAmount           string                `gorm:"size:32" json:"payAmount"`
	RefundAmount        string                `gorm:"size:32" json:"refundAmount"`
	AftersaleType       string                `gorm:"size:64" json:"aftersaleType"`
	Reason              string                `gorm:"size:256" json:"reason"`
	Status              string                `gorm:"size:64;index" json:"status"`
	TimeoutText         string                `gorm:"size:128" json:"timeoutText"`
	Dispute             string                `gorm:"size:64" json:"dispute"`
	Logistics           string                `gorm:"size:256" json:"logistics"`
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
