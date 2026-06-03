package model

import "time"

type SeedancePortrait struct {
	Id         uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId     uint      `gorm:"column:user_id;type:int(11);not null;index;comment:用户ID" json:"user_id"`
	AssetId    string    `gorm:"column:asset_id;type:varchar(200);not null;comment:Seedance素材ID" json:"asset_id"`
	AssetUrl   string    `gorm:"column:asset_url;type:varchar(500);not null;comment:asset://URL" json:"asset_url"`
	PreviewUrl string    `gorm:"column:preview_url;type:varchar(500);not null;comment:预览图URL" json:"preview_url"`
	Name       string    `gorm:"column:name;type:varchar(200);comment:人像名称" json:"name"`
	CreatedAt  time.Time `gorm:"column:created_at;comment:创建时间" json:"created_at"`
}
