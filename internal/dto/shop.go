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
	LastSyncAt         string `json:"lastSyncAt,omitempty"`
	LastSeenAt         string `json:"lastSeenAt,omitempty"`
	SyncRequested      bool   `json:"syncRequested"`
	PendingTicketCount int    `json:"pendingTicketCount,omitempty"`
	Remark             string `json:"remark,omitempty"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type PluginHeartbeatResult struct {
	ShopItem
	SyncNow         bool `json:"syncNow"`
	SyncIntervalSec int  `json:"syncIntervalSec"`
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
	ProductTags         string   `json:"productTags,omitempty"`
	Tags                string   `json:"tags,omitempty"`
	Qty                 int      `json:"qty"`
	BuyQty              int      `json:"buyQty"`
	PayAmount           string   `json:"payAmount"`
	RefundAmount        string   `json:"refundAmount"`
	AftersaleType       string   `json:"aftersaleType"`
	Reason              string   `json:"reason"`
	Status              string   `json:"status"`
	TimeoutText         string   `json:"timeoutText,omitempty"`
	TimeoutAction       string   `json:"timeoutAction,omitempty"`
	DeadlineAt          string   `json:"deadlineAt,omitempty"`
	RemainSeconds       int      `json:"remainSeconds"`
	Dispute             string   `json:"dispute,omitempty"`
	Logistics           string   `json:"logistics,omitempty"`
	ReturnLogisticsNo   string   `json:"returnLogisticsNo,omitempty"`
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
	ProductTags         string   `json:"productTags"`
	Tags                string   `json:"tags"`
	Qty                 int      `json:"qty"`
	BuyQty              int      `json:"buyQty"`
	PayAmount           string   `json:"payAmount"`
	RefundAmount        string   `json:"refundAmount"`
	AftersaleType       string   `json:"aftersaleType"`
	Reason              string   `json:"reason"`
	Status              string   `json:"status"`
	TimeoutText         string   `json:"timeoutText"`
	Dispute             string   `json:"dispute"`
	Logistics           string   `json:"logistics"`
	ReturnLogisticsNo   string   `json:"returnLogisticsNo"`
	ApplyTime           string   `json:"applyTime"`
	CardKeys            []string `json:"cardKeys"`
	RawJSON             string   `json:"rawJson"`
}

type PluginSyncReturn struct {
	PlatformAftersaleID string `json:"platformAftersaleId"`
	OrderNo             string `json:"orderNo"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage"`
	SKU                 string `json:"sku"`
	Qty                 int    `json:"qty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	Logistics           string `json:"logistics"`
	LogisticsNo         string `json:"logisticsNo"`
	Carrier             string `json:"carrier"`
	ReturnLocation      string `json:"returnLocation"`
	ShipTime            string `json:"shipTime"`
	ApplyTime           string `json:"applyTime"`
	ReturnTime          string `json:"returnTime"`
	TrackJSON           string `json:"trackJson"`
	RawJSON             string `json:"rawJson"`
}

type PluginSyncInput struct {
	PluginAuthInput
	PlatformShopID   string              `json:"platformShopId"`
	PlatformShopName string              `json:"platformShopName"`
	Cards            []PluginSyncCard    `json:"cards"`
	Tickets []PluginSyncTicket  `json:"tickets"`
	Returns *[]PluginSyncReturn `json:"returns"`
}

type PluginSyncResult struct {
	ShopID      uint64 `json:"shopId"`
	CardCount   int    `json:"cardCount"`
	TicketCount int    `json:"ticketCount"`
	ReturnCount int    `json:"returnCount"`
	LastSyncAt  string `json:"lastSyncAt"`
}

type ReturnPackageItem struct {
	ID                  uint64 `json:"id"`
	ShopID              uint64 `json:"shopId"`
	ShopName            string `json:"shopName"`
	PlatformAftersaleID string `json:"platformAftersaleId"`
	OrderNo             string `json:"orderNo"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage,omitempty"`
	SKU                 string `json:"sku"`
	Qty                 int    `json:"qty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	Logistics           string `json:"logistics,omitempty"`
	LogisticsNo         string `json:"logisticsNo"`
	Carrier             string `json:"carrier,omitempty"`
	ReturnLocation      string `json:"returnLocation"`
	ShipTime            string `json:"shipTime,omitempty"`
	ApplyTime           string `json:"applyTime,omitempty"`
	ReturnTime          string `json:"returnTime,omitempty"`
	SyncedAt            string `json:"syncedAt"`
}

type ReturnListQuery struct {
	ShopID     uint64
	Keyword    string
	ReturnFrom string
	ReturnTo   string
	ApplyFrom  string
	ApplyTo    string
	Page       int
	PageSize   int
}
