package dto

type IssueOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type IssueMeta struct {
	ProblemTypes  []IssueOption `json:"problemTypes"`
	HandleMethods []IssueOption `json:"handleMethods"`
	Statuses      []IssueOption `json:"statuses"`
}

type IssueLookupResult struct {
	OrderNo             string `json:"orderNo"`
	PlatformOrderID     string `json:"platformOrderId,omitempty"`
	PlatformAftersaleID string `json:"platformAftersaleId,omitempty"`
	ShopID              uint64 `json:"shopId,omitempty"`
	ShopName            string `json:"shopName"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage,omitempty"`
	SkuSpecs            string `json:"skuSpecs,omitempty"`
	BuyerName           string `json:"buyerName,omitempty"`
	BuyerPhone          string `json:"buyerPhone,omitempty"`
	BuyerAddress        string `json:"buyerAddress,omitempty"`
	HasAftersale        bool   `json:"hasAftersale"`
	AftersaleStatus     string `json:"aftersaleStatus,omitempty"`
	AftersaleType       string `json:"aftersaleType,omitempty"`
	Logistics           string `json:"logistics,omitempty"`
	ReturnLogisticsNo   string `json:"returnLogisticsNo,omitempty"`
	ShipLogisticsNo     string `json:"shipLogisticsNo,omitempty"`
	Source              string `json:"source"` // ticket / order / both
}

type IssueItem struct {
	ID                  uint64  `json:"id"`
	ShopID              uint64  `json:"shopId,omitempty"`
	ShopName            string  `json:"shopName"`
	OrderNo             string  `json:"orderNo,omitempty"`
	PlatformOrderID     string  `json:"platformOrderId,omitempty"`
	PlatformAftersaleID string  `json:"platformAftersaleId,omitempty"`
	ProductTitle        string  `json:"productTitle"`
	ProductImage        string  `json:"productImage,omitempty"`
	SkuSpecs            string  `json:"skuSpecs,omitempty"`
	BuyerName           string  `json:"buyerName,omitempty"`
	BuyerPhone          string  `json:"buyerPhone,omitempty"`
	BuyerAddress        string  `json:"buyerAddress,omitempty"`
	HasAftersale        bool    `json:"hasAftersale"`
	AftersaleStatus     string  `json:"aftersaleStatus,omitempty"`
	AftersaleType       string  `json:"aftersaleType,omitempty"`
	Logistics           string  `json:"logistics,omitempty"`
	ReturnLogisticsNo   string  `json:"returnLogisticsNo,omitempty"`
	ShipLogisticsNo     string  `json:"shipLogisticsNo,omitempty"`
	ProblemType         string  `json:"problemType"`
	ProblemTypeCustom   string  `json:"problemTypeCustom,omitempty"`
	ProblemTypeLabel    string  `json:"problemTypeLabel"`
	HandleMethod        string  `json:"handleMethod,omitempty"`
	HandleMethodCustom  string  `json:"handleMethodCustom,omitempty"`
	HandleMethodLabel   string  `json:"handleMethodLabel,omitempty"`
	ProblemNote         string  `json:"problemNote,omitempty"`
	HandleNote          string  `json:"handleNote,omitempty"`
	Status              string  `json:"status"`
	StatusLabel         string  `json:"statusLabel"`
	OperatorName        string  `json:"operatorName,omitempty"`
	CompletedAt         *string `json:"completedAt,omitempty"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

type IssueListQuery struct {
	ShopID      uint64
	Status      string
	ProblemType string
	Keyword     string
	Page        int
	PageSize    int
}

type IssueUpsertRequest struct {
	ShopID              uint64 `json:"shopId"`
	ShopName            string `json:"shopName"`
	OrderNo             string `json:"orderNo"`
	PlatformOrderID     string `json:"platformOrderId"`
	PlatformAftersaleID string `json:"platformAftersaleId"`
	ProductTitle        string `json:"productTitle"`
	ProductImage        string `json:"productImage"`
	SkuSpecs            string `json:"skuSpecs"`
	BuyerName           string `json:"buyerName"`
	BuyerPhone          string `json:"buyerPhone"`
	BuyerAddress        string `json:"buyerAddress"`
	HasAftersale        bool   `json:"hasAftersale"`
	AftersaleStatus     string `json:"aftersaleStatus"`
	AftersaleType       string `json:"aftersaleType"`
	Logistics           string `json:"logistics"`
	ReturnLogisticsNo   string `json:"returnLogisticsNo"`
	ShipLogisticsNo     string `json:"shipLogisticsNo"`
	ProblemType         string `json:"problemType"`
	ProblemTypeCustom   string `json:"problemTypeCustom"`
	HandleMethod        string `json:"handleMethod"`
	HandleMethodCustom  string `json:"handleMethodCustom"`
	ProblemNote         string `json:"problemNote"`
	HandleNote          string `json:"handleNote"`
	Status              string `json:"status"`
}

type IssueStatusRequest struct {
	Status string `json:"status"`
}
