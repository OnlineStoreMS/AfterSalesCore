package dto

type UnboxingCreateInput struct {
	TrackingNo string `json:"trackingNo" binding:"required"`
	Remark     string `json:"remark"`
}

type UnboxingCompleteInput struct {
	VideoDurationSec int    `json:"videoDurationSec"`
	Remark           string `json:"remark"`
}

type UnboxingPhotoDetail struct {
	ID          uint64 `json:"id"`
	PhotoURL    string `json:"photoUrl"`
	IssueRemark string `json:"issueRemark"`
	SortOrder   int    `json:"sortOrder"`
	CreatedAt   string `json:"createdAt"`
}

type UnboxingDetail struct {
	ID               uint64                `json:"id"`
	TrackingNo       string                `json:"trackingNo"`
	VideoURL         string                `json:"videoUrl"`
	VideoSize        int64                 `json:"videoSize"`
	VideoDurationSec int                   `json:"videoDurationSec"`
	VideoMimeType    string                `json:"videoMimeType"`
	Status           string                `json:"status"`
	Remark           string                `json:"remark"`
	OperatorID       uint64                `json:"operatorId"`
	OperatorName     string                `json:"operatorName"`
	CreatedAt        string                `json:"createdAt"`
	Photos           []UnboxingPhotoDetail `json:"photos"`
}

type UnboxingListItem struct {
	ID               uint64 `json:"id"`
	TrackingNo       string `json:"trackingNo"`
	Status           string `json:"status"`
	VideoURL         string `json:"videoUrl,omitempty"`
	VideoDurationSec int    `json:"videoDurationSec"`
	PhotoCount       int    `json:"photoCount"`
	OperatorName     string `json:"operatorName"`
	CreatedAt        string `json:"createdAt"`
}

type UnboxingDownloadResp struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}
