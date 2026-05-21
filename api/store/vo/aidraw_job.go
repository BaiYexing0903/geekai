package vo

type AiDrawJob struct {
	Id        uint   `json:"id"`
	UserId    uint   `json:"user_id"`
	Mode      string `json:"mode"`
	Prompt    string `json:"prompt"`
	ImgURL    string `json:"img_url"`
	OrgURL    string `json:"org_url"`
	Publish   int    `json:"publish"`
	Power     int    `json:"power"`
	Progress  int    `json:"progress"`
	ErrMsg    string `json:"err_msg"`
	CreatedAt int64  `json:"created_at"`
}
