package handler

// * +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// * Copyright 2023 The Geek-AI Authors. All rights reserved.
// * Use of this source code is governed by a Apache-2.0 license
// * that can be found in the LICENSE file.
// * @Author yangjian102621@163.com
// * +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

import (
	"geekai/core"
	"geekai/core/types"
	"geekai/store/model"
	"geekai/utils"
	"geekai/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConfigHandler struct {
	BaseHandler
}

func NewConfigHandler(app *core.AppServer, db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

// RegisterRoutes 注册路由
func (h *ConfigHandler) RegisterRoutes() {
	group := h.App.Engine.Group("/api/config/")

	// 无需授权的接口
	group.GET("get", h.Get)
	group.GET("license", h.License)
}

var publicConfigKeys = map[string]bool{
	"system":  true,
	"notice":  true,
	"license": true,
	"mj":      true,
	"sd":      true,
	"suno":    true,
	"footer":  true,
}

// Get 获取指定的系统配置
func (h *ConfigHandler) Get(c *gin.Context) {
	key := c.Query("key")
	if !publicConfigKeys[key] {
		resp.ERROR(c, "无权访问该配置")
		return
	}

	var config model.Config
	res := h.DB.Where("name", key).First(&config)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	var value map[string]any
	err := utils.JsonDecode(config.Value, &value)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, value)
}

// License 获取 License 配置
func (h *ConfigHandler) License(c *gin.Context) {
	resp.SUCCESS(c, types.LicenseConfig{
		DeCopy:  true,
		UserNum: 0,
	})
}
