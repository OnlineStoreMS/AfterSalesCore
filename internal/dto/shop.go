package dto

type ShopCreateInput struct {
	Name     string `json:"name" binding:"required"`
	Platform string `json:"platform"`
	Remark   string `json:"remark"`
}

type ShopUpdateInput struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type ShopItem struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	PlatformLabel    string `json:"platformLabel"`
	BindCode         string `json:"bindCode"`
	PluginStatus     string `json:"pluginStatus"`
	PluginAvailable  bool   `json:"pluginAvailable"`
	PlatformShopID   string `json:"platformShopId,omitempty"`
	PlatformShopName string `json:"platformShopName,omitempty"`
	LastSyncAt       string `json:"lastSyncAt,omitempty"`
	LastSeenAt       string `json:"lastSeenAt,omitempty"`
	Remark           string `json:"remark,omitempty"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type FilterCardItem struct {
	GroupName string `json:"groupName"`
	CardKey   string `json:"cardKey"`
	CardLabel string `json:"cardLabel"`
	Count     int    `json:"count"`
	SortOrder int    `json:"sortOrder"`
}

type TicketItem struct {
	ID                  uint64   `json:"id"`
	PlatformAftersaleID string   `json:"platformAftersaleId"`
	OrderNo             string   `json:"orderNo"`
	ProductTitle        string   `json:"productTitle"`
	ProductImage        string   `json:"productImage,omitempty"`
	SKU                 string   `json:"sku"`
	Qty                 int      `json:"qty"`
	PayAmount           string   `json:"payAmount"`
	RefundAmount        string   `json:"refundAmount"`
	AftersaleType       string   `json:"aftersaleType"`
	Reason              string   `json:"reason"`
	Status              string   `json:"status"`
	TimeoutText         string   `json:"timeoutText,omitempty"`
	Dispute             string   `json:"dispute,omitempty"`
	Logistics           string   `json:"logistics,omitempty"`
	ApplyTime           string   `json:"applyTime,omitempty"`
	CardKeys            []string `json:"cardKeys"`
	SyncedAt            string   `json:"syncedAt"`
}

type ShopWorkbench struct {
	Shop       ShopItem         `json:"shop"`
	Cards      []FilterCardItem `json:"cards"`
	Tickets    []TicketItem     `json:"tickets"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	LastSyncAt string           `json:"lastSyncAt,omitempty"`
}

type PluginBindInput struct {
	BindCode string `json:"bindCode" binding:"required"`
}

type PluginBindResult struct {
	ShopID       uint64 `json:"shopId"`
	ShopName     string `json:"shopName"`
	Platform     string `json:"platform"`
	PluginKey    string `json:"pluginKey"`
	PluginSecret string `json:"pluginSecret"`
}

type PluginAuthInput struct {
	PluginKey    string `json:"pluginKey"`
	PluginSecret string `json:"pluginSecret"`
}

type PluginHeartbeatInput struct {
	PluginAuthInput
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
}

type PluginSyncCard struct {
	GroupName string `json:"groupName"`
	CardKey   string `json:"cardKey"`
	CardLabel string `json:"cardLabel"`
	Count     int    `json:"count"`
	SortOrder int    `json:"sortOrder"`
}

type PluginSyncTicket struct {
	PlatformAftersaleID string   `json:"platformAftersaleId"`
	OrderNo             string   `json:"orderNo"`
	ProductTitle        string   `json:"productTitle"`
	ProductImage        string   `json:"productImage"`
	SKU                 string   `json:"sku"`
	Qty                 int      `json:"qty"`
	PayAmount           string   `json:"payAmount"`
	RefundAmount        string   `json:"refundAmount"`
	AftersaleType       string   `json:"aftersaleType"`
	Reason              string   `json:"reason"`
	Status              string   `json:"status"`
	TimeoutText         string   `json:"timeoutText"`
	Dispute             string   `json:"dispute"`
	Logistics           string   `json:"logistics"`
	ApplyTime           string   `json:"applyTime"`
	CardKeys            []string `json:"cardKeys"`
	RawJSON             string   `json:"rawJson"`
}

type PluginSyncInput struct {
	PluginAuthInput
	PlatformShopID   string             `json:"platformShopId"`
	PlatformShopName string             `json:"platformShopName"`
	Cards            []PluginSyncCard   `json:"cards"`
	Tickets          []PluginSyncTicket `json:"tickets"`
}

type PluginSyncResult struct {
	ShopID      uint64 `json:"shopId"`
	CardCount   int    `json:"cardCount"`
	TicketCount int    `json:"ticketCount"`
	LastSyncAt  string `json:"lastSyncAt"`
}
