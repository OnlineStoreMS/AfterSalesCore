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
	ID                 uint64 `json:"id"`
	Name               string `json:"name"`
	Platform           string `json:"platform"`
	PlatformLabel      string `json:"platformLabel"`
	BindCode           string `json:"bindCode"`
	PluginStatus       string `json:"pluginStatus"`
	PluginAvailable    bool   `json:"pluginAvailable"`
	PlatformShopID     string `json:"platformShopId,omitempty"`
	PlatformShopName   string `json:"platformShopName,omitempty"`
	LastSyncAt         string `json:"lastSyncAt,omitempty"`
	LastSeenAt         string `json:"lastSeenAt,omitempty"`
	NextSyncAt         string `json:"nextSyncAt,omitempty"`
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

type PluginSetting struct {
	PluginSyncIntervalMin int `json:"pluginSyncIntervalMin"`
}

type FilterCardItem struct {
	GroupName string `json:"groupName"`
	CardKey   string `json:"cardKey"`
	CardLabel string `json:"cardLabel"`
	Count     int    `json:"count"`
	SortOrder int    `json:"sortOrder"`
}

type TicketItem struct {
	ID                   uint64   `json:"id"`
	PlatformAftersaleID  string   `json:"platformAftersaleId"`
	OrderNo              string   `json:"orderNo"`
	ProductTitle         string   `json:"productTitle"`
	ProductImage         string   `json:"productImage,omitempty"`
	SKU                  string   `json:"sku"`
	ProductTags          string   `json:"productTags,omitempty"`
	Tags                 string   `json:"tags,omitempty"`
	Qty                  int      `json:"qty"`
	BuyQty               int      `json:"buyQty"`
	PayAmount            string   `json:"payAmount"`
	RefundAmount         string   `json:"refundAmount"`
	AftersaleType        string   `json:"aftersaleType"`
	Reason               string   `json:"reason"`
	Status               string   `json:"status"`
	TimeoutText          string   `json:"timeoutText,omitempty"`
	TimeoutAction        string   `json:"timeoutAction,omitempty"`
	DeadlineAt           string   `json:"deadlineAt,omitempty"`
	RemainSeconds        int      `json:"remainSeconds"`
	Dispute              string   `json:"dispute,omitempty"`
	Logistics            string   `json:"logistics,omitempty"`
	LogisticsBuyerStatus string   `json:"logisticsBuyerStatus,omitempty"`
	LogisticsShipStatus  string   `json:"logisticsShipStatus,omitempty"`
	NeedIntercept        bool     `json:"needIntercept,omitempty"`
	ReturnLogisticsNo    string   `json:"returnLogisticsNo,omitempty"`
	ShipLogisticsNo      string   `json:"shipLogisticsNo,omitempty"`
	ApplyTime            string   `json:"applyTime,omitempty"`
	CardKeys             []string `json:"cardKeys"`
	SyncedAt             string   `json:"syncedAt"`
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
	ShipLogisticsNo     string   `json:"shipLogisticsNo"`
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
	BuyQty              int    `json:"buyQty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	OrderInfo           string `json:"orderInfo"`
	AftersaleInfo       string `json:"aftersaleInfo"`
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
	PlatformShopID   string                     `json:"platformShopId"`
	PlatformShopName string                     `json:"platformShopName"`
	Cards            []PluginSyncCard           `json:"cards"`
	Tickets          []PluginSyncTicket         `json:"tickets"`
	Returns          *[]PluginSyncReturn        `json:"returns"`
	ShippedRefunds   *[]PluginSyncShippedRefund `json:"shippedRefunds"`
	ServiceOrders    *[]PluginSyncServiceOrder  `json:"serviceOrders"`
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

type PluginSyncShippedRefund struct {
	PlatformAftersaleID string `json:"platformAftersaleId"`
	OrderNo             string `json:"orderNo"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage"`
	SKU                 string `json:"sku"`
	ProductTags         string `json:"productTags"`
	Tags                string `json:"tags"`
	Qty                 int    `json:"qty"`
	BuyQty              int    `json:"buyQty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	OrderInfo           string `json:"orderInfo"`
	AftersaleInfo       string `json:"aftersaleInfo"`
	Logistics           string `json:"logistics"`
	LogisticsStatus     string `json:"logisticsStatus"`
	LogisticsNo         string `json:"logisticsNo"`
	Carrier             string `json:"carrier"`
	ShipTime            string `json:"shipTime"`
	TrackJSON           string `json:"trackJson"`
	ApplyTime           string `json:"applyTime"`
	RawJSON             string `json:"rawJson"`
}

type PluginSyncResult struct {
	ShopID             uint64 `json:"shopId"`
	CardCount          int    `json:"cardCount"`
	TicketCount        int    `json:"ticketCount"`
	ReturnCount        int    `json:"returnCount"`
	ShippedRefundCount int    `json:"shippedRefundCount"`
	ServiceOrderCount  int    `json:"serviceOrderCount"`
	LastSyncAt         string `json:"lastSyncAt"`
}

type ServiceOrderItem struct {
	ID                uint64 `json:"id"`
	ShopID            uint64 `json:"shopId"`
	ShopName          string `json:"shopName"`
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

type ServiceOrderListQuery struct {
	ShopID    uint64
	StatusTab string
	Keyword   string
	Page      int
	PageSize  int
}

type ServiceOrderListResult struct {
	List     []ServiceOrderItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Tabs     []ServiceTabCount  `json:"tabs"`
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
	BuyQty              int    `json:"buyQty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	OrderInfo           string `json:"orderInfo,omitempty"`
	AftersaleInfo       string `json:"aftersaleInfo,omitempty"`
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

type ShippedRefundItem struct {
	ID                  uint64           `json:"id"`
	ShopID              uint64           `json:"shopId"`
	ShopName            string           `json:"shopName"`
	PlatformAftersaleID string           `json:"platformAftersaleId"`
	OrderNo             string           `json:"orderNo"`
	ProductTitle        string           `json:"productTitle"`
	ProductImage        string           `json:"productImage,omitempty"`
	SKU                 string           `json:"sku"`
	ProductTags         string           `json:"productTags,omitempty"`
	Tags                string           `json:"tags,omitempty"`
	Qty                 int              `json:"qty"`
	BuyQty              int              `json:"buyQty"`
	PayAmount           string           `json:"payAmount"`
	RefundAmount        string           `json:"refundAmount"`
	AftersaleType       string           `json:"aftersaleType"`
	Reason              string           `json:"reason"`
	Status              string           `json:"status"`
	OrderInfo           string           `json:"orderInfo,omitempty"`
	AftersaleInfo       string           `json:"aftersaleInfo,omitempty"`
	Logistics           string           `json:"logistics,omitempty"`
	LogisticsStatus     string           `json:"logisticsStatus,omitempty"`
	LogisticsNo         string           `json:"logisticsNo,omitempty"`
	Carrier             string           `json:"carrier,omitempty"`
	ShipTime            string           `json:"shipTime,omitempty"`
	Tracks              []LogisticsTrack `json:"tracks,omitempty"`
	Alert               bool             `json:"alert"`
	ApplyTime           string           `json:"applyTime,omitempty"`
	SyncedAt            string           `json:"syncedAt"`
}

type LogisticsTrack struct {
	Date   string `json:"date"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Text   string `json:"text"`
}

type ShippedRefundListQuery struct {
	ShopID    uint64
	Keyword   string
	Status    string
	AlertOnly bool
	ApplyFrom string
	ApplyTo   string
	Page      int
	PageSize  int
}

type InterceptItem struct {
	ID                  uint64 `json:"id"`
	ShopID              uint64 `json:"shopId"`
	ShopName            string `json:"shopName"`
	Source              string `json:"source"`
	NeedIntercept       bool   `json:"needIntercept"`
	AwaitPickup         bool   `json:"awaitPickup"`
	PlatformAftersaleID string `json:"platformAftersaleId"`
	OrderNo             string `json:"orderNo"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage,omitempty"`
	SKU                 string `json:"sku"`
	Qty                 int    `json:"qty"`
	BuyQty              int    `json:"buyQty"`
	PayAmount           string `json:"payAmount"`
	RefundAmount        string `json:"refundAmount"`
	AftersaleType       string `json:"aftersaleType"`
	Reason              string `json:"reason"`
	Status              string `json:"status"`
	Logistics           string `json:"logistics,omitempty"`
	LogisticsStatus     string `json:"logisticsStatus,omitempty"`
	LogisticsNo         string `json:"logisticsNo,omitempty"`
	ShipLogisticsNo     string `json:"shipLogisticsNo,omitempty"`
	ReturnLogisticsNo   string `json:"returnLogisticsNo,omitempty"`
	Carrier             string `json:"carrier,omitempty"`
	ApplyTime           string `json:"applyTime,omitempty"`
	SyncedAt            string `json:"syncedAt"`
}

type InterceptListQuery struct {
	ShopID   uint64
	Keyword  string
	Page     int
	PageSize int
}
