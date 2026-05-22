package handler

import (
	"fmt"
	"geekai/core"
	"geekai/core/middleware"
	"geekai/core/types"
	"geekai/service"
	"geekai/service/moderation"
	"geekai/service/seedance"
	"geekai/store/model"
	"geekai/store/vo"
	"geekai/utils"
	"geekai/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SeedanceHandler struct {
	BaseHandler
	seedanceService   *seedance.Service
	userService       *service.UserService
	moderationManager *moderation.ServiceManager
}

func NewSeedanceHandler(app *core.AppServer, seedanceService *seedance.Service, db *gorm.DB, userService *service.UserService, moderationManager *moderation.ServiceManager) *SeedanceHandler {
	return &SeedanceHandler{
		BaseHandler:       BaseHandler{App: app, DB: db},
		seedanceService:   seedanceService,
		userService:       userService,
		moderationManager: moderationManager,
	}
}

func (h *SeedanceHandler) RegisterRoutes() {
	group := h.App.Engine.Group("/api/seedance/")
	group.Use(middleware.UserAuthMiddleware(h.App.Config.Session.SecretKey, h.App.Redis))
	{
		group.POST("task", h.CreateTask)
		group.GET("power-config", h.GetPowerConfig)
		group.POST("jobs", h.Jobs)
		group.GET("remove", h.Remove)
		group.GET("retry", h.Retry)
	}
}

type SeedanceTaskRequest struct {
	TaskType      string   `json:"task_type" binding:"required"`
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	FirstFrameURL string   `json:"first_frame_url"`
	LastFrameURL  string   `json:"last_frame_url"`
	ImageUrls     []string `json:"image_urls"`
	VideoUrls     []string `json:"video_urls"`
	AudioUrls     []string `json:"audio_urls"`
	RefVideoURL   string   `json:"ref_video_url"`
	RefImageURL   string   `json:"ref_image_url"`
	GenerateAudio bool     `json:"generate_audio"`
	Resolution    string   `json:"resolution"`
	Ratio         string   `json:"ratio"`
	Duration      int      `json:"duration"`
	Watermark     bool     `json:"watermark"`
	AssetId       string   `json:"asset_id"`
}

func (h *SeedanceHandler) CreateTask(c *gin.Context) {
	var req SeedanceTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 文本审核
	if h.App.SysConfig.Moderation.Enable && req.Prompt != "" {
		moderationResult, err := h.moderationManager.GetService().Moderate(req.Prompt)
		if err != nil {
			logger.Error("failed to moderate content: ", err)
		}
		if moderationResult.Flagged {
			m := model.Moderation{
				UserId: h.GetLoginUserId(c),
				Source: types.ModerationSourceSeedance,
				Input:  req.Prompt,
				Result: utils.JsonEncode(moderationResult),
			}
			_ = h.DB.Create(&m).Error
			resp.ERROR(c, "当前创作内容包含敏感词，请重新输入！")
			return
		}
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 获取配置
	config := h.App.SysConfig.Seedance

	// 确定模型和每秒单价
	modelId := config.ModelFast
	var priceMap map[string]int
	if req.Model == "standard" {
		modelId = config.ModelStd
		priceMap = config.Power.VipPrice
	} else {
		priceMap = config.Power.FastPrice
	}
	if priceMap == nil {
		priceMap = make(map[string]int)
	}
	resolution := req.Resolution
	if resolution == "" {
		resolution = "720p"
	}
	perSecond := priceMap[resolution]
	if perSecond <= 0 {
		perSecond = priceMap["720p"]
		if perSecond <= 0 {
			perSecond = 1
		}
	}

	// 计算算力：模型每秒单价 × 时长
	if perSecond <= 0 {
		perSecond = 1
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 5
	}
	powerCost := perSecond * duration

	// 确定模式
	var mode model.SDTaskMode
	var content []seedance.ContentItem

	switch req.TaskType {
	case "text_to_video":
		mode = model.SDModeTextToVideo
		content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})

	case "image_to_video_first":
		mode = model.SDModeImageToVideoFirst
		if req.FirstFrameURL == "" {
			resp.ERROR(c, "请上传首帧图片")
			return
		}
		content = append(content, seedance.ContentItem{
			Type: "image_url", ImageURL: &seedance.URLField{URL: req.FirstFrameURL}, Role: "first_frame",
		})
		if req.Prompt != "" {
			content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})
		}

	case "image_to_video_dual":
		mode = model.SDModeImageToVideoDual
		if req.FirstFrameURL == "" || req.LastFrameURL == "" {
			resp.ERROR(c, "请上传首帧和尾帧图片")
			return
		}
		content = append(content,
			seedance.ContentItem{Type: "image_url", ImageURL: &seedance.URLField{URL: req.FirstFrameURL}, Role: "first_frame"},
			seedance.ContentItem{Type: "image_url", ImageURL: &seedance.URLField{URL: req.LastFrameURL}, Role: "last_frame"},
		)
		if req.Prompt != "" {
			content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})
		}

	case "multimodal_ref":
		mode = model.SDModeMultimodalRef
		for _, url := range req.ImageUrls {
			content = append(content, seedance.ContentItem{Type: "image_url", ImageURL: &seedance.URLField{URL: url}, Role: "reference_image"})
		}
		for _, url := range req.VideoUrls {
			content = append(content, seedance.ContentItem{Type: "video_url", VideoURL: &seedance.URLField{URL: url}, Role: "reference_video"})
		}
		for _, url := range req.AudioUrls {
			content = append(content, seedance.ContentItem{Type: "audio_url", AudioURL: &seedance.URLField{URL: url}, Role: "reference_audio"})
		}
		if req.Prompt != "" {
			content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})
		}

	case "edit_video":
		mode = model.SDModeEditVideo
		if req.RefVideoURL == "" {
			resp.ERROR(c, "请上传参考视频")
			return
		}
		content = append(content, seedance.ContentItem{Type: "video_url", VideoURL: &seedance.URLField{URL: req.RefVideoURL}, Role: "reference_video"})
		if req.RefImageURL != "" {
			content = append(content, seedance.ContentItem{Type: "image_url", ImageURL: &seedance.URLField{URL: req.RefImageURL}, Role: "reference_image"})
		}
		content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})

	case "extend_video":
		mode = model.SDModeExtendVideo
		if len(req.VideoUrls) == 0 {
			resp.ERROR(c, "请上传参考视频")
			return
		}
		for _, url := range req.VideoUrls {
			content = append(content, seedance.ContentItem{Type: "video_url", VideoURL: &seedance.URLField{URL: url}, Role: "reference_video"})
		}
		if req.Prompt != "" {
			content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})
		}

	case "virtual_avatar":
		mode = model.SDModeVirtualAvatar
		if req.AssetId == "" {
			resp.ERROR(c, "请选择虚拟人像")
			return
		}
		content = append(content, seedance.ContentItem{
			Type: "image_url", ImageURL: &seedance.URLField{URL: "asset://" + req.AssetId}, Role: "reference_image",
		})
		if req.Prompt != "" {
			content = append(content, seedance.ContentItem{Type: "text", Text: req.Prompt})
		}

	default:
		resp.ERROR(c, "不支持的任务类型")
		return
	}

	if user.Power < powerCost {
		resp.ERROR(c, fmt.Sprintf("算力不足，需要%d算力", powerCost))
		return
	}

	// 设置默认值
	if req.Resolution == "" {
		req.Resolution = "720p"
	}
	if req.Ratio == "" {
		req.Ratio = "16:9"
	}
	if req.Duration == 0 {
		req.Duration = 5
	}

	taskReq := &seedance.CreateJobReq{
		Mode:          mode,
		Model:         modelId,
		Prompt:        req.Prompt,
		Content:       content,
		GenerateAudio: req.GenerateAudio,
		Resolution:    req.Resolution,
		Ratio:         req.Ratio,
		Duration:      req.Duration,
		Watermark:     req.Watermark,
		Power:         powerCost,
	}

	job, err := h.seedanceService.CreateTask(user.Id, taskReq)
	if err != nil {
		logger.Errorf("create seedance task failed: %v", err)
		resp.ERROR(c, "创建任务失败")
		return
	}

	h.userService.DecreasePower(user.Id, powerCost, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  "seedance",
		Remark: fmt.Sprintf("Seedance视频生成，任务ID：%d", job.Id),
	})

	resp.SUCCESS(c, job)
}

func (h *SeedanceHandler) GetPowerConfig(c *gin.Context) {
	config := h.App.SysConfig.Seedance
	resp.SUCCESS(c, config.Power)
}

func (h *SeedanceHandler) Jobs(c *gin.Context) {
	userId := h.GetLoginUserId(c)

	var req struct {
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
		Filter   string `json:"filter"`
		Ids      []uint `json:"ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var jobs []model.SeedanceJob
	var total int64
	query := h.DB.Model(&model.SeedanceJob{}).Where("user_id = ?", userId)

	if len(req.Ids) > 0 {
		query = query.Where("id IN (?)", req.Ids)
	}

	if err := query.Count(&total).Error; err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(req.PageSize).Find(&jobs).Error; err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	var jobVos []vo.SeedanceJob
	for _, job := range jobs {
		var jobVo vo.SeedanceJob
		err := utils.CopyObject(job, &jobVo)
		if err != nil {
			continue
		}
		jobVo.CreatedAt = job.CreatedAt.Unix()
		jobVo.UpdatedAt = job.UpdatedAt.Unix()
		jobVos = append(jobVos, jobVo)
	}
	resp.SUCCESS(c, vo.NewPage(total, req.Page, req.PageSize, jobVos))
}

func (h *SeedanceHandler) Remove(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	jobId := h.GetInt(c, "id", 0)
	if jobId == 0 {
		resp.ERROR(c, "参数错误")
		return
	}

	job, err := h.seedanceService.GetJob(uint(jobId))
	if err != nil {
		resp.ERROR(c, "任务不存在")
		return
	}
	if job.UserId != user.Id {
		resp.ERROR(c, "无权限操作")
		return
	}

	if job.Status == model.SDStatusQueued || job.Status == model.SDStatusRunning {
		resp.ERROR(c, "正在运行中的任务不能删除，否则无法退回算力")
		return
	}

	tx := h.DB.Begin()
	if err := tx.Where("id = ? AND user_id = ?", jobId, user.Id).Delete(&model.SeedanceJob{}).Error; err != nil {
		logger.Errorf("delete seedance job failed: %v", err)
		resp.ERROR(c, "删除任务失败")
		tx.Rollback()
		return
	}

	// 失败任务删除后退回算力
	if job.Status == model.SDStatusFailed && job.Power > 0 {
		err = h.userService.IncreasePower(user.Id, job.Power, model.PowerLog{
			Type:   types.PowerRefund,
			Model:  "seedance",
			Remark: fmt.Sprintf("删除任务，退回%d算力", job.Power),
		})
		if err != nil {
			resp.ERROR(c, "退回算力失败")
			tx.Rollback()
			return
		}
	}

	tx.Commit()
	resp.SUCCESS(c, gin.H{})
}

func (h *SeedanceHandler) Retry(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	jobId := h.GetInt(c, "id", 0)
	if jobId == 0 {
		resp.ERROR(c, "参数错误")
		return
	}

	job, err := h.seedanceService.GetJob(uint(jobId))
	if err != nil {
		resp.ERROR(c, "任务不存在")
		return
	}
	if job.UserId != userId {
		resp.ERROR(c, "无权限操作")
		return
	}
	if job.Status != model.SDStatusFailed {
		resp.ERROR(c, "只有失败的任务才能重试")
		return
	}

	if err := h.seedanceService.UpdateJobStatus(uint(jobId), model.SDStatusQueued, ""); err != nil {
		logger.Errorf("reset seedance job status failed: %v", err)
		resp.ERROR(c, "重置任务状态失败")
		return
	}

	if err := h.seedanceService.PushTaskToQueue(uint(jobId)); err != nil {
		logger.Errorf("push retry seedance task to queue failed: %v", err)
		resp.ERROR(c, "推送重试任务失败")
		return
	}

	resp.SUCCESS(c, gin.H{"message": "重试任务已提交"})
}
