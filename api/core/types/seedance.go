package types

type SeedanceConfig struct {
	ApiURL      string         `json:"api_url"`
	BearerToken string         `json:"bearer_token"`
	ModelFast   string         `json:"model_fast"`
	ModelStd    string         `json:"model_std"`
	Power       SeedancePower  `json:"power"`
}

type SeedancePower struct {
	FastPrice map[string]int `json:"fast_price"` // 分辨率 → 每秒算力，如 {"480p":3, "720p":5, "1080p":8}
	VipPrice  map[string]int `json:"vip_price"`  // 分辨率 → 每秒算力
}
