package vo

import "geekai/store/model"

type SeedanceJob struct {
	Id           uint                `json:"id"`
	UserId       uint                `json:"user_id"`
	TaskId       string              `json:"task_id"`
	Type         model.SDTaskMode    `json:"type"`
	Model        string              `json:"model"`
	Prompt       string              `json:"prompt"`
	TaskParams   string              `json:"task_params"`
	CoverURL     string              `json:"cover_url"`
	VideoURL     string              `json:"video_url"`
	LastFrameURL string              `json:"last_frame_url"`
	RawData      string              `json:"raw_data"`
	Status       model.SDTaskStatus  `json:"status"`
	ErrMsg       string              `json:"err_msg"`
	Power        int                 `json:"power"`
	Progress     int                 `json:"progress"`
	CreatedAt    int64               `json:"created_at"`
	UpdatedAt    int64               `json:"updated_at"`
}
