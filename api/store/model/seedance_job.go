package model

import "time"

type SDTaskStatus string

const (
	SDStatusQueued    SDTaskStatus = "queued"
	SDStatusRunning   SDTaskStatus = "running"
	SDStatusSucceeded SDTaskStatus = "succeeded"
	SDStatusFailed    SDTaskStatus = "failed"
	SDStatusExpired   SDTaskStatus = "expired"
)

type SDTaskMode string

const (
	SDModeTextToVideo       SDTaskMode = "text_to_video"
	SDModeImageToVideoFirst SDTaskMode = "image_to_video_first"
	SDModeImageToVideoDual  SDTaskMode = "image_to_video_dual"
	SDModeMultimodalRef     SDTaskMode = "multimodal_ref"
	SDModeEditVideo         SDTaskMode = "edit_video"
	SDModeExtendVideo       SDTaskMode = "extend_video"
	SDModeVirtualAvatar     SDTaskMode = "virtual_avatar"
)

type SeedanceJob struct {
	Id            uint         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId        uint         `gorm:"column:user_id;type:int(11);not null;index;comment:用户ID" json:"user_id"`
	TaskId        string       `gorm:"column:task_id;type:varchar(200);not null;index;comment:API任务ID" json:"task_id"`
	Type          SDTaskMode   `gorm:"column:type;type:varchar(50);not null;comment:视频模式" json:"type"`
	Model         string       `gorm:"column:model;type:varchar(100);comment:模型endpoint ID" json:"model"`
	Prompt        string       `gorm:"column:prompt;type:text;comment:提示词" json:"prompt"`
	TaskParams    string       `gorm:"column:task_params;type:text;comment:任务参数JSON" json:"task_params"`
	CoverURL      string       `gorm:"column:cover_url;type:varchar(1024);comment:封面图URL" json:"cover_url"`
	VideoURL      string       `gorm:"column:video_url;type:varchar(1024);comment:视频URL" json:"video_url"`
	LastFrameURL  string       `gorm:"column:last_frame_url;type:varchar(1024);comment:末帧图片URL" json:"last_frame_url"`
	RawData       string       `gorm:"column:raw_data;type:text;comment:原始API响应" json:"raw_data"`
	Status        SDTaskStatus `gorm:"column:status;type:varchar(20);default:'queued';comment:任务状态" json:"status"`
	ErrMsg        string       `gorm:"column:err_msg;type:varchar(1024);comment:错误信息" json:"err_msg"`
	Power         int          `gorm:"column:power;type:int(11);default:0;comment:消耗算力" json:"power"`
	Progress      int          `gorm:"column:progress;type:int;default:0;comment:进度" json:"progress"`
	CreatedAt     time.Time    `gorm:"column:created_at;type:datetime;not null" json:"created_at"`
	UpdatedAt     time.Time    `gorm:"column:updated_at;type:datetime;not null" json:"updated_at"`
}

func (SeedanceJob) TableName() string {
	return "geekai_seedance_jobs"
}
