package admin

import (
	"fmt"
	"geekai/core"
	"geekai/core/types"
	"geekai/handler"
	"geekai/service"
	"geekai/service/oss"
	"geekai/service/seedance"
	"geekai/store/model"
	"geekai/store/vo"
	"geekai/utils"
	"geekai/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SeedanceHandler struct {
	handler.BaseHandler
	seedanceClient *seedance.Client
	userService    *service.UserService
	uploader       *oss.UploaderManager
}

func NewSeedanceHandler(app *core.AppServer, db *gorm.DB, seedanceClient *seedance.Client, userService *service.UserService, uploader *oss.UploaderManager) *SeedanceHandler {
	return &SeedanceHandler{
		BaseHandler:    handler.BaseHandler{App: app, DB: db},
		seedanceClient: seedanceClient,
		userService:    userService,
		uploader:       uploader,
	}
}

func (h *SeedanceHandler) RegisterRoutes() {
	rg := h.App.Engine.Group("/api/admin/seedance/")
	rg.GET("/jobs", h.Jobs)
	rg.GET("/jobs/:id", h.JobDetail)
	rg.POST("/jobs/remove", h.BatchRemove)
	rg.GET("/stats", h.Stats)
	rg.POST("/config/update", h.UpdateConfig)
}

func (h *SeedanceHandler) Jobs(c *gin.Context) {
	var req struct {
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
		UserId   uint   `form:"user_id"`
		Type     string `form:"type"`
		Status   string `form:"status"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	query := h.DB.Model(&model.SeedanceJob{})
	if req.UserId > 0 {
		query = query.Where("user_id = ?", req.UserId)
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	query.Count(&total)

	var jobs []model.SeedanceJob
	offset := (req.Page - 1) * req.PageSize
	query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&jobs)

	var jobVos []vo.SeedanceJob
	for _, job := range jobs {
		var jobVo vo.SeedanceJob
		_ = utils.CopyObject(job, &jobVo)
		jobVo.CreatedAt = job.CreatedAt.Unix()
		jobVo.UpdatedAt = job.UpdatedAt.Unix()
		jobVos = append(jobVos, jobVo)
	}

	resp.SUCCESS(c, vo.NewPage(total, req.Page, req.PageSize, jobVos))
}

func (h *SeedanceHandler) JobDetail(c *gin.Context) {
	id := c.Param("id")
	var job model.SeedanceJob
	if err := h.DB.First(&job, id).Error; err != nil {
		resp.ERROR(c, "任务不存在")
		return
	}
	resp.SUCCESS(c, job)
}

func (h *SeedanceHandler) BatchRemove(c *gin.Context) {
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var jobs []model.SeedanceJob
	h.DB.Where("id IN (?)", req.Ids).Find(&jobs)

	for _, job := range jobs {
		if job.VideoURL != "" {
			_ = h.uploader.GetUploadHandler().Delete(job.VideoURL)
		}
		if job.CoverURL != "" {
			_ = h.uploader.GetUploadHandler().Delete(job.CoverURL)
		}
		if job.LastFrameURL != "" {
			_ = h.uploader.GetUploadHandler().Delete(job.LastFrameURL)
		}
		if job.Status == model.SDStatusFailed && job.Power > 0 {
			_ = h.userService.IncreasePower(job.UserId, job.Power, model.PowerLog{
				Type:   types.PowerRefund,
				Model:  "seedance",
				Remark: fmt.Sprintf("管理员删除任务，退回%d算力", job.Power),
			})
		}
	}

	h.DB.Where("id IN (?)", req.Ids).Delete(&model.SeedanceJob{})
	resp.SUCCESS(c, gin.H{})
}

func (h *SeedanceHandler) Stats(c *gin.Context) {
	type StatResult struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var stats []StatResult
	h.DB.Model(&model.SeedanceJob{}).Select("status, COUNT(*) as count").Group("status").Find(&stats)

	result := map[string]any{
		"total":      int64(0),
		"completed":  int64(0),
		"processing": int64(0),
		"failed":     int64(0),
		"pending":    int64(0),
	}
	for _, stat := range stats {
		result["total"] = result["total"].(int64) + stat.Count
		switch stat.Status {
		case "queued":
			result["pending"] = stat.Count
		case "running":
			result["processing"] = stat.Count
		case "succeeded":
			result["completed"] = stat.Count
		case "failed":
			result["failed"] = stat.Count
		}
	}
	resp.SUCCESS(c, result)
}

func (h *SeedanceHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		ApiURL      string `json:"api_url"`
		BearerToken string `json:"bearer_token"`
		ModelFast   string `json:"model_fast"`
		ModelStd    string `json:"model_std"`
		Power       struct {
			TextToVideo       int `json:"text_to_video"`
			ImageToVideoFirst int `json:"image_to_video_first"`
			ImageToVideoDual  int `json:"image_to_video_dual"`
			MultimodalRef     int `json:"multimodal_ref"`
			EditVideo         int `json:"edit_video"`
			ExtendVideo       int `json:"extend_video"`
			VirtualAvatar     int `json:"virtual_avatar"`
		} `json:"power"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	config := types.SeedanceConfig{
		ApiURL:      req.ApiURL,
		BearerToken: req.BearerToken,
		ModelFast:   req.ModelFast,
		ModelStd:    req.ModelStd,
		Power:       types.SeedancePower(req.Power),
	}

	configValue := utils.JsonEncode(config)
	var sysConfig model.Config
	h.DB.Where("name", types.ConfigKeySeedance).First(&sysConfig)
	sysConfig.Name = types.ConfigKeySeedance
	sysConfig.Value = configValue
	if sysConfig.Id > 0 {
		h.DB.Where("name", types.ConfigKeySeedance).Updates(&sysConfig)
	} else {
		h.DB.Create(&sysConfig)
	}

	h.seedanceClient.UpdateConfig(config)
	h.App.SysConfig.Seedance = config

	resp.SUCCESS(c, gin.H{})
}
