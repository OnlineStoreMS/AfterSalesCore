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

type ServiceOrderItem struct {
	ID                uint64 `json:"id"`
	PlatformServiceID string `json:"platformServiceId"`
	OrderNo           string `json:"orderNo"`
	ProductTitle      string `json:"productTitle"`
	ProductImage      string `json:"productImage,omitempty"`
	ProductContent    string `json:"productContent,omitempty"`
	BuyerNick         string `json:"buyerNick,omitempty"`
	CreateSource      string `json:"createSource,omitempty"`
	BusinessType      string `json:"businessType,omitempty"`
	OrderType         string `json:"orderType,omitempty"`
	Tags              string `json:"tags,omitempty"`
	StatusTab         string `json:"statusTab"`
	Status            string `json:"status"`
	TimeoutText       string `json:"timeoutText,omitempty"`
	TimeoutAction     string `json:"timeoutAction,omitempty"`
	DeadlineAt        string `json:"deadlineAt,omitempty"`
	RemainSeconds     int    `json:"remainSeconds"`
	Detail            string `json:"detail,omitempty"`
	Solution          string `json:"solution,omitempty"`
	LastLog           string `json:"lastLog,omitempty"`
	LastLogTime       string `json:"lastLogTime,omitempty"`
	CreateTime        string `json:"createTime,omitempty"`
	SyncedAt          string `json:"syncedAt"`
}

type ServiceTabCount struct {
	StatusTab string `json:"statusTab"`
	Count     int64  `json:"count"`
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

type PluginSyncInput struct {
	PluginAuthInput
	PlatformShopID   string                   `json:"platformShopId"`
	PlatformShopName string                   `json:"platformShopName"`
	Cards            []PluginSyncCard         `json:"cards"`
	Tickets          []PluginSyncTicket       `json:"tickets"`
	ServiceOrders    []PluginSyncServiceOrder `json:"serviceOrders"`
}

type PluginSyncServiceOrder struct {
	PlatformServiceID string `json:"platformServiceId"`
	OrderNo           string `json:"orderNo"`
	ProductTitle      string `json:"productTitle"`
	ProductImage      string `json:"productImage"`
	ProductContent    string `json:"productContent"`
	BuyerNick         string `json:"buyerNick"`
	CreateSource      string `json:"createSource"`
	BusinessType      string `json:"businessType"`
	OrderType         string `json:"orderType"`
	Tags              string `json:"tags"`
	StatusTab         string `json:"statusTab"`
	Status            string `json:"status"`
	TimeoutText       string `json:"timeoutText"`
	DelayEndTime      int64  `json:"delayEndTime"`
	DelayTimeLeft     int64  `json:"delayTimeLeft"`
	Detail            string `json:"detail"`
	Solution          string `json:"solution"`
	LastLog           string `json:"lastLog"`
	LastLogTime       string `json:"lastLogTime"`
	CreateTime        string `json:"createTime"`
	RawJSON           string `json:"rawJson"`
}

type PluginSyncResult struct {
	ShopID            uint64 `json:"shopId"`
	CardCount         int    `json:"cardCount"`
	TicketCount       int    `json:"ticketCount"`
	ServiceOrderCount int    `json:"serviceOrderCount"`
	LastSyncAt        string `json:"lastSyncAt"`
}
