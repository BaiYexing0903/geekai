package seedance

import "geekai/store/model"

// --- Seedance REST API 请求/响应类型 ---

type ContentItem struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *URLField       `json:"image_url,omitempty"`
	VideoURL *URLField       `json:"video_url,omitempty"`
	AudioURL *URLField       `json:"audio_url,omitempty"`
	Role     string          `json:"role,omitempty"`
}

type URLField struct {
	URL string `json:"url"`
}

// CreateTaskReq POST /doubao/create 请求
type CreateTaskReq struct {
	Model         string        `json:"model"`
	Content       []ContentItem `json:"content"`
	GenerateAudio bool          `json:"generate_audio,omitempty"`
	Resolution    string        `json:"resolution,omitempty"`
	Ratio         string        `json:"ratio,omitempty"`
	Duration      int           `json:"duration,omitempty"`
	Watermark     bool          `json:"watermark,omitempty"`
}

// CreateTaskResp POST /doubao/create 响应
type CreateTaskResp struct {
	ID      string `json:"id"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// QueryTaskResp POST /doubao/get_result 响应
type QueryTaskResp struct {
	ID         string       `json:"id"`
	Model      string       `json:"model,omitempty"`
	Status     string       `json:"status"`
	Error      *TaskError   `json:"error,omitempty"`
	CreatedAt  int64        `json:"created_at,omitempty"`
	UpdatedAt  int64        `json:"updated_at,omitempty"`
	Content    *TaskContent `json:"content,omitempty"`
	Seed       int          `json:"seed,omitempty"`
	Resolution string       `json:"resolution,omitempty"`
	Ratio      string       `json:"ratio,omitempty"`
	Duration   int          `json:"duration,omitempty"`
	Usage      *TaskUsage   `json:"usage,omitempty"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TaskContent struct {
	VideoURL     string `json:"video_url"`
	FileURL      string `json:"file_url"`
	LastFrameURL string `json:"last_frame_url"`
}

type TaskUsage struct {
	CompletionTokens int `json:"completion_tokens"`
}

// --- 内部任务创建请求（handler → service）---

type CreateJobReq struct {
	Mode          model.SDTaskMode `json:"mode"`
	Model         string           `json:"model"`
	Prompt        string           `json:"prompt"`
	Content       []ContentItem    `json:"content"`
	GenerateAudio bool             `json:"generate_audio"`
	Resolution    string           `json:"resolution"`
	Ratio         string           `json:"ratio"`
	Duration      int              `json:"duration"`
	Watermark     bool             `json:"watermark"`
	Power         int              `json:"power"`
}

type MediaAssetFilter struct {
	Field string          `json:"Field"`
	Op    string          `json:"Op"`
	Conds MediaAssetConds `json:"Conds"`
}

type MediaAssetConds struct {
	StrValues []string `json:"StrValues"`
}

type ListMediaAssetGroupReq struct {
	PageNum   int                `json:"PageNum"`
	PageSize  int                `json:"PageSize"`
	SortBy    string             `json:"SortBy"`
	SortOrder string             `json:"SortOrder"`
	Filters   []MediaAssetFilter `json:"Filters"`
}

type ListMediaAssetGroupResp struct {
	Result  MediaAssetGroupResult `json:"Result"`
	Code    string                `json:"code,omitempty"`
	Message string                `json:"message,omitempty"`
}

type MediaAssetGroupResult struct {
	Items      []MediaAssetGroupItem `json:"Items"`
	TotalCount int                   `json:"TotalCount"`
	PageNum    int                   `json:"PageNum"`
	PageSize   int                   `json:"PageSize"`
}

type MediaAssetGroupItem struct {
	AssetGroup MediaAssetGroup `json:"AssetGroup"`
}

type MediaAssetGroup struct {
	SID            string                 `json:"SID"`
	Title          string                 `json:"Title"`
	Description    string                 `json:"Description"`
	Metadata       MediaAssetMetadata     `json:"Metadata"`
	Content        MediaAssetContent      `json:"Content"`
	Score          float64                `json:"Score"`
	AdditionalInfo map[string]interface{} `json:"AdditionalInfo"`
}

type MediaAssetMetadata struct {
	Country    string `json:"Country"`
	Age        int    `json:"Age"`
	Gender     string `json:"Gender"`
	Occupation string `json:"Occupation"`
	Type       string `json:"Type"`
}

type MediaAssetContent struct {
	Image []MediaAssetImage `json:"Image"`
}

type MediaAssetImage struct {
	AssetID string `json:"AssetID"`
	URL     string `json:"URL"`
}
