package types

type SeedanceConfig struct {
	ApiURL      string         `json:"api_url"`
	BearerToken string         `json:"bearer_token"`
	ModelFast   string         `json:"model_fast"`
	ModelStd    string         `json:"model_std"`
	Power       SeedancePower  `json:"power"`
}

type SeedancePower struct {
	TextToVideo       int `json:"text_to_video"`
	ImageToVideoFirst int `json:"image_to_video_first"`
	ImageToVideoDual  int `json:"image_to_video_dual"`
	MultimodalRef     int `json:"multimodal_ref"`
	EditVideo         int `json:"edit_video"`
	ExtendVideo       int `json:"extend_video"`
	VirtualAvatar     int `json:"virtual_avatar"`
}
