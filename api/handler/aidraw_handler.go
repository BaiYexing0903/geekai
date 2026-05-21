package handler

import (
	"fmt"
	"geekai/core"
	"geekai/core/middleware"
	"geekai/core/types"
	"geekai/service"
	"geekai/service/aidraw"
	"geekai/service/moderation"
	"geekai/service/oss"
	"geekai/store/model"
	"geekai/store/vo"
	"geekai/utils"
	"geekai/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AiDrawHandler struct {
	BaseHandler
	aidrawService    *aidraw.Service
	uploader         *oss.UploaderManager
	userService      *service.UserService
	moderationManager *moderation.ServiceManager
}

func NewAiDrawHandler(app *core.AppServer, db *gorm.DB, service *aidraw.Service, manager *oss.UploaderManager, userService *service.UserService, moderationManager *moderation.ServiceManager) *AiDrawHandler {
	return &AiDrawHandler{
		aidrawService:    service,
		uploader:         manager,
		userService:      userService,
		moderationManager: moderationManager,
		BaseHandler: BaseHandler{
			App: app,
			DB:  db,
		},
	}
}

func (h *AiDrawHandler) RegisterRoutes() {
	group := h.App.Engine.Group("/api/aidraw/")

	// 公开接口
	group.GET("imgWall", h.ImgWall)
	group.GET("models", h.GetModels)

	// 需要用户授权的接口
	group.Use(middleware.UserAuthMiddleware(h.App.Config.Session.SecretKey, h.App.Redis))
	{
		group.POST("image", h.Image)
		group.GET("jobs", h.JobList)
		group.GET("remove", h.Remove)
		group.GET("publish", h.Publish)
	}
}

func (h *AiDrawHandler) Image(c *gin.Context) {
	var data types.AiDrawTask
	if err := c.ShouldBindJSON(&data); err != nil || data.Prompt == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 文本审核
	if h.App.SysConfig.Moderation.Enable {
		moderationResult, err := h.moderationManager.GetService().Moderate(data.Prompt)
		if err != nil {
			logger.Error("failed to moderate content: ", err)
		}
		if moderationResult.Flagged {
			moderation := model.Moderation{
				UserId: h.GetLoginUserId(c),
				Source: types.ModerationSourceAiDraw,
				Input:  data.Prompt,
				Result: utils.JsonEncode(moderationResult),
			}
			_ = h.DB.Create(&moderation).Error
			resp.ERROR(c, "当前创作内容包含敏感词，提示词未通过文本审核，请重新输入！")
			return
		}
	}

	var chatModel model.ChatModel
	if res := h.DB.Where("id = ?", data.ModelId).First(&chatModel); res.Error != nil {
		resp.ERROR(c, "模型不存在")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	if user.Power < chatModel.Power {
		resp.ERROR(c, "当前用户剩余算力不足以完成本次绘画！")
		return
	}

	idValue, _ := c.Get(types.LoginUserID)
	userId := utils.IntValue(utils.InterfaceToString(idValue), 0)

	mode := data.Mode
	if mode == "" {
		mode = "text_to_image"
	}

	task := types.AiDrawTask{
		UserId:           uint(userId),
		ModelId:          chatModel.Id,
		ModelName:        chatModel.Value,
		Mode:             mode,
		Prompt:           data.Prompt,
		Images:           data.Images,
		AspectRatio:      data.AspectRatio,
		ImageSize:        data.ImageSize,
		Quality:          data.Quality,
		Size:             data.Size,
		TranslateModelId: h.App.SysConfig.Base.AssistantModelId,
		Power:            chatModel.Power,
	}

	job := model.AiDrawJob{
		UserId:   uint(userId),
		Mode:     mode,
		Prompt:   data.Prompt,
		Power:    chatModel.Power,
		TaskInfo: utils.JsonEncode(task),
	}
	res := h.DB.Create(&job)
	if res.Error != nil {
		resp.ERROR(c, "error with save job: "+res.Error.Error())
		return
	}

	task.Id = job.Id
	h.aidrawService.PushTask(task)

	err = h.userService.DecreasePower(user.Id, chatModel.Power, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  chatModel.Value,
		Remark: fmt.Sprintf("AI绘画提示词：%s", utils.CutWords(task.Prompt, 10)),
	})
	if err != nil {
		resp.ERROR(c, "error with decrease power: "+err.Error())
		return
	}
	resp.SUCCESS(c)
}

func (h *AiDrawHandler) ImgWall(c *gin.Context) {
	page := h.GetInt(c, "page", 0)
	pageSize := h.GetInt(c, "page_size", 0)
	err, jobs := h.getData(true, 0, page, pageSize, true)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, jobs)
}

func (h *AiDrawHandler) JobList(c *gin.Context) {
	finish := h.GetBool(c, "finish")
	userId := h.GetLoginUserId(c)
	page := h.GetInt(c, "page", 0)
	pageSize := h.GetInt(c, "page_size", 0)
	publish := h.GetBool(c, "publish")

	err, jobs := h.getData(finish, userId, page, pageSize, publish)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, jobs)
}

func (h *AiDrawHandler) getData(finish bool, userId uint, page int, pageSize int, publish bool) (error, vo.Page) {
	session := h.DB.Session(&gorm.Session{})
	if finish {
		session = session.Where("progress >= ?", 100).Order("id DESC")
	} else {
		session = session.Where("progress < ?", 100).Order("id ASC")
	}
	if userId > 0 {
		session = session.Where("user_id = ?", userId)
	}
	if publish {
		session = session.Where("publish", publish)
	}
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		session = session.Offset(offset).Limit(pageSize)
	}

	var total int64
	session.Model(&model.AiDrawJob{}).Count(&total)

	var items []model.AiDrawJob
	res := session.Find(&items)
	if res.Error != nil {
		return res.Error, vo.Page{}
	}

	var jobs = make([]vo.AiDrawJob, 0)
	for _, item := range items {
		var job vo.AiDrawJob
		err := utils.CopyObject(item, &job)
		if err != nil {
			continue
		}
		job.CreatedAt = item.CreatedAt.Unix()
		jobs = append(jobs, job)
	}

	return nil, vo.NewPage(total, page, pageSize, jobs)
}

func (h *AiDrawHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	userId := h.GetLoginUserId(c)
	var job model.AiDrawJob
	if res := h.DB.Where("id = ? AND user_id = ?", id, userId).First(&job); res.Error != nil {
		resp.ERROR(c, "记录不存在")
		return
	}

	err := h.DB.Delete(&job).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	_ = h.uploader.GetUploadHandler().Delete(job.ImgURL)
	resp.SUCCESS(c)
}

func (h *AiDrawHandler) Publish(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	userId := h.GetLoginUserId(c)
	action := h.GetBool(c, "action")

	err := h.DB.Model(&model.AiDrawJob{Id: uint(id), UserId: userId}).UpdateColumn("publish", action).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c)
}

func (h *AiDrawHandler) GetModels(c *gin.Context) {
	var models []model.ChatModel
	err := h.DB.Where("type", "img").Where("enabled", true).Find(&models).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	var modelVos []vo.ChatModel
	for _, v := range models {
		var modelVo vo.ChatModel
		err := utils.CopyObject(v, &modelVo)
		if err != nil {
			continue
		}
		modelVo.Id = v.Id
		modelVos = append(modelVos, modelVo)
	}

	resp.SUCCESS(c, modelVos)
}
