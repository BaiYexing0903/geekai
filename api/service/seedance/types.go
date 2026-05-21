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
