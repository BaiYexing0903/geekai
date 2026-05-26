package types

// JimengConfig 即梦AI配置
type JimengConfig struct {
	AccessKey   string      `json:"access_key"`
	SecretKey   string      `json:"secret_key"`
	ApiUrl      string      `json:"api_url"`
	BearerToken string      `json:"bearer_token"`
	Power       JimengPower `json:"power"`
}

// JimengPower 即梦AI算力配置
type JimengPower struct {
	TextToImage  int `json:"text_to_image"`
	JimengV4T2i  int `json:"jimeng_v4_t2i"`
	JimengV4I2i  int `json:"jimeng_v4_i2i"`
	ImageToImage int `json:"image_to_image"`
	ImageEdit    int `json:"image_edit"`
	ImageEffects int `json:"image_effects"`
	TextToVideo  int `json:"text_to_video"`
	ImageToVideo int `json:"image_to_video"`
}
