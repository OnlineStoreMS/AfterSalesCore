package dto

type EdgeRecordCreateInput struct {
	Type        string `json:"type" binding:"required"`
	TrackingNo  string `json:"trackingNo" binding:"required"`
	Remark      string `json:"remark"`
}

type EdgeRecordCompleteInput struct {
	Remark string `json:"remark"`
}

type EdgeRecordListItem struct {
	ID               int64    `json:"id"`
	EdgeID           string   `json:"edgeId"`
	EdgeName         string   `json:"edgeName,omitempty"`
	Type             string   `json:"type"`
	TrackingNo       string   `json:"trackingNo"`
	Status           string   `json:"status"`
	VideoURL         string   `json:"videoUrl,omitempty"`
	VideoDurationSec int      `json:"videoDurationSec,omitempty"`
	PhotoCount       int      `json:"photoCount"`
	Remark           string   `json:"remark,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	CompletedAt      string   `json:"completedAt,omitempty"`
}

type EdgeRecordPhotoDetail struct {
	URL string `json:"url"`
}

type EdgeRecordDetail struct {
	ID               int64                   `json:"id"`
	EdgeID           string                  `json:"edgeId"`
	EdgeName         string                  `json:"edgeName,omitempty"`
	Type             string                  `json:"type"`
	TrackingNo       string                  `json:"trackingNo"`
	Status           string                  `json:"status"`
	VideoURL         string                  `json:"videoUrl,omitempty"`
	VideoSize        int64                   `json:"videoSize,omitempty"`
	VideoDurationSec int                     `json:"videoDurationSec,omitempty"`
	Remark           string                  `json:"remark,omitempty"`
	Photos           []EdgeRecordPhotoDetail `json:"photos"`
	CreatedAt        string                  `json:"createdAt"`
	UpdatedAt        string                  `json:"updatedAt"`
	CompletedAt      string                  `json:"completedAt,omitempty"`
}

type EdgeRecordDownloadResp struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

type EdgeRecordBatchDeleteInput struct {
	IDs []int64 `json:"ids" binding:"required"`
}
