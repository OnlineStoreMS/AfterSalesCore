package dto

type EdgeDeviceCreateInput struct {
	EdgeID  string `json:"edgeId" binding:"required"`
	Name    string `json:"name" binding:"required"`
	BaseURL string `json:"baseUrl"`
	Remark  string `json:"remark"`
}

type EdgeDeviceUpdateInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Remark  string `json:"remark"`
}

type EdgeDeviceItem struct {
	ID         uint64 `json:"id"`
	EdgeID     string `json:"edgeId"`
	Name       string `json:"name"`
	BaseURL    string `json:"baseUrl"`
	Status     string `json:"status"`
	LastSeenAt string `json:"lastSeenAt,omitempty"`
	Remark     string `json:"remark,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}
