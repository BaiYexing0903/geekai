package jimeng

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	logger2 "geekai/logger"
	"geekai/service/oss"
	"geekai/store"
	"geekai/store/model"
	"geekai/utils"

	"github.com/go-redis/redis/v8"
)

var logger = logger2.GetLogger()

// Service 即梦服务（合并了消费者功能）
type Service struct {
	db        *gorm.DB
	redis     *redis.Client
	taskQueue *store.RedisQueue
	client    *Client
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	uploader  *oss.UploaderManager
}

// NewService 创建即梦服务
func NewService(db *gorm.DB, redisCli *redis.Client, uploader *oss.UploaderManager, client *Client) *Service {
	taskQueue := store.NewRedisQueue("JimengTaskQueue", redisCli)
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:        db,
		redis:     redisCli,
		taskQueue: taskQueue,
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
		running:   false,
		uploader:  uploader,
	}
}

// Start 启动服务（包含消费者）
func (s *Service) Start() {
	if s.running {
		return
	}
	logger.Info("Starting Jimeng service and task consumer...")
	s.running = true
	go s.consumeTasks()
	go s.pollTaskStatus()
}

// Stop 停止服务
func (s *Service) Stop() {
	if !s.running {
		return
	}
	logger.Info("Stopping Jimeng service and task consumer...")
	s.running = false
	s.cancel()
}

// consumeTasks 消费任务
func (s *Service) consumeTasks() {
	for {
		select {
		case <-s.ctx.Done():
			logger.Info("Jimeng task consumer stopped")
			return
		default:
			s.processNextTask()
		}
	}
}

// processNextTask 处理下一个任务
func (s *Service) processNextTask() {
	var jobId uint
	if err := s.taskQueue.LPop(&jobId); err != nil {
		time.Sleep(time.Second)
		return
	}

	logger.Infof("Processing Jimeng task: job_id=%d", jobId)

	if err := s.ProcessTask(jobId); err != nil {
		logger.Errorf("process jimeng task failed: job_id=%d, error=%v", jobId, err)
		s.UpdateJobStatus(jobId, model.JMTaskStatusFailed, err.Error())
	} else {
		logger.Infof("Jimeng task processed successfully: job_id=%d", jobId)
	}
}

// CreateTask 创建任务
func (s *Service) CreateTask(userId uint, req *CreateTaskRequest) (*model.JimengJob, error) {
	taskId := utils.RandString(20)

	paramsJson, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("marshal task params failed: %w", err)
	}

	job := &model.JimengJob{
		UserId:     userId,
		TaskId:     taskId,
		Type:       req.Type,
		ReqKey:     req.ReqKey,
		Prompt:     req.Prompt,
		TaskParams: string(paramsJson),
		Status:     model.JMTaskStatusInQueue,
		Power:      req.Power,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("create jimeng job failed: %w", err)
	}

	if err := s.taskQueue.RPush(job.Id); err != nil {
		return nil, fmt.Errorf("push jimeng task to queue failed: %w", err)
	}

	return job, nil
}

// ProcessTask 处理任务
func (s *Service) ProcessTask(jobId uint) error {
	var job model.JimengJob
	if err := s.db.First(&job, jobId).Error; err != nil {
		return fmt.Errorf("get jimeng job failed: %w", err)
	}

	if err := s.UpdateJobStatus(job.Id, model.JMTaskStatusGenerating, ""); err != nil {
		return fmt.Errorf("update job status failed: %w", err)
	}

	req, err := s.buildTaskRequest(&job)
	if err != nil {
		return s.handleTaskError(job.Id, fmt.Sprintf("build task request failed: %v", err))
	}

	logger.Infof("提交即梦任务: %+v", req)

	var taskId string
	var rawData []byte

	if job.Type == model.JMTaskTypeJimengV4T2i || job.Type == model.JMTaskTypeJimengV4I2i {
		v4Req := &V4CreateRequest{
			ReqKey:      req.ReqKey,
			Prompt:      req.Prompt,
			Width:       req.Width,
			Height:      req.Height,
			Scale:       req.Scale,
			ForceSingle: true,
		}
		resp, err := s.client.SubmitV4Task(v4Req)
		if err != nil {
			return s.handleTaskError(job.Id, fmt.Sprintf("submit v4 task failed: %v", err))
		}
		if resp.Code != 10000 {
			return s.handleTaskError(job.Id, fmt.Sprintf("submit v4 task failed: %s", mapV4ErrorMessage(resp.Code, resp.Message)))
		}
		taskId = resp.Data.TaskId
		rawData, _ = json.Marshal(resp)
	} else {
		resp, err := s.client.SubmitTask(req)
		if err != nil {
			return s.handleTaskError(job.Id, fmt.Sprintf("submit task failed: %v", err))
		}
		if resp.Code != 10000 {
			return s.handleTaskError(job.Id, fmt.Sprintf("submit task failed: %s", resp.Message))
		}
		taskId = resp.Data.TaskId
		rawData, _ = json.Marshal(resp)
	}

	if err := s.db.Model(&model.JimengJob{}).Where("id = ?", job.Id).Updates(map[string]any{
		"task_id":    taskId,
		"raw_data":   string(rawData),
		"updated_at": time.Now(),
	}).Error; err != nil {
		logger.Errorf("update jimeng job task_id failed: %v", err)
	}

	return nil
}

// buildTaskRequest 构建任务请求（统一的参数解析）
func (s *Service) buildTaskRequest(job *model.JimengJob) (*SubmitTaskRequest, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(job.TaskParams), &params); err != nil {
		return nil, fmt.Errorf("parse task params failed: %w", err)
	}

	req := &SubmitTaskRequest{
		ReqKey: job.ReqKey,
		Prompt: job.Prompt,
	}

	switch job.Type {
	case model.JMTaskTypeTextToImage:
		s.setTextToImageParams(req, params)
	case model.JMTaskTypeJimengV4T2i:
		s.setJimengV4T2iParams(req, params)
	case model.JMTaskTypeJimengV4I2i:
		s.setJimengV4I2iParams(req, params)
	case model.JMTaskTypeImageToImage:
		s.setImageToImageParams(req, params)
	case model.JMTaskTypeImageEdit:
		s.setImageEditParams(req, params)
	case model.JMTaskTypeImageEffects:
		s.setImageEffectsParams(req, params)
	case model.JMTaskTypeTextToVideo:
		s.setTextToVideoParams(req, params)
	case model.JMTaskTypeImageToVideo:
		s.setImageToVideoParams(req, params)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", job.Type)
	}

	return req, nil
}

// setTextToImageParams 设置文生图参数
func (s *Service) setTextToImageParams(req *SubmitTaskRequest, params map[string]any) {
	if seed, ok := params["seed"]; ok {
		if seedVal, err := strconv.ParseInt(fmt.Sprintf("%.0f", seed), 10, 64); err == nil {
			req.Seed = seedVal
		}
	}
	if scale, ok := params["scale"]; ok {
		if scaleVal, ok := scale.(float64); ok {
			req.Scale = scaleVal
		}
	}
	if width, ok := params["width"]; ok {
		if widthVal, ok := width.(float64); ok {
			req.Width = int(widthVal)
		}
	}
	if height, ok := params["height"]; ok {
		if heightVal, ok := height.(float64); ok {
			req.Height = int(heightVal)
		}
	}
	if usePreLlm, ok := params["use_pre_llm"]; ok {
		if usePreLlmVal, ok := usePreLlm.(bool); ok {
			req.UsePreLLM = usePreLlmVal
		}
	}
}

// setImageToImageParams 设置图生图参数
func (s *Service) setImageToImageParams(req *SubmitTaskRequest, params map[string]any) {
	if imageInput, ok := params["image_input"].(string); ok {
		req.ImageInput = imageInput
	}
	if gpen, ok := params["gpen"]; ok {
		if gpenVal, ok := gpen.(float64); ok {
			req.Gpen = gpenVal
		}
	}
	if skin, ok := params["skin"]; ok {
		if skinVal, ok := skin.(float64); ok {
			req.Skin = skinVal
		}
	}
	if skinUnifi, ok := params["skin_unifi"]; ok {
		if skinUnifiVal, ok := skinUnifi.(float64); ok {
			req.SkinUnifi = skinUnifiVal
		}
	}
	if genMode, ok := params["gen_mode"].(string); ok {
		req.GenMode = genMode
	}
	s.setCommonParams(req, params)
}

// setImageEditParams 设置图像编辑参数
func (s *Service) setImageEditParams(req *SubmitTaskRequest, params map[string]any) {
	if imageUrls, ok := params["image_urls"].([]any); ok {
		for _, url := range imageUrls {
			if urlStr, ok := url.(string); ok {
				req.ImageUrls = append(req.ImageUrls, urlStr)
			}
		}
	}
	if binaryData, ok := params["binary_data_base64"].([]any); ok {
		for _, data := range binaryData {
			if dataStr, ok := data.(string); ok {
				req.BinaryDataBase64 = append(req.BinaryDataBase64, dataStr)
			}
		}
	}
	if scale, ok := params["scale"]; ok {
		if scaleVal, ok := scale.(float64); ok {
			req.Scale = scaleVal
		}
	}
	s.setCommonParams(req, params)
}

// setImageEffectsParams 设置图像特效参数
func (s *Service) setImageEffectsParams(req *SubmitTaskRequest, params map[string]any) {
	if imageInput1, ok := params["image_input1"].(string); ok {
		req.ImageInput1 = imageInput1
	}
	if templateId, ok := params["template_id"].(string); ok {
		req.TemplateId = templateId
	}
	if width, ok := params["width"]; ok {
		if widthVal, ok := width.(float64); ok {
			req.Width = int(widthVal)
		}
	}
	if height, ok := params["height"]; ok {
		if heightVal, ok := height.(float64); ok {
			req.Height = int(heightVal)
		}
	}
}

// setTextToVideoParams 设置文生视频参数
func (s *Service) setTextToVideoParams(req *SubmitTaskRequest, params map[string]any) {
	if aspectRatio, ok := params["aspect_ratio"].(string); ok {
		req.AspectRatio = aspectRatio
	}
	s.setCommonParams(req, params)
}

// setImageToVideoParams 设置图生视频参数
func (s *Service) setImageToVideoParams(req *SubmitTaskRequest, params map[string]any) {
	s.setImageEditParams(req, params)
	if aspectRatio, ok := params["aspect_ratio"].(string); ok {
		req.AspectRatio = aspectRatio
	}
}

// setJimengV4T2iParams 设置即梦4.0文生图参数
func (s *Service) setJimengV4T2iParams(req *SubmitTaskRequest, params map[string]any) {
	if width, ok := params["width"]; ok {
		if widthVal, ok := width.(float64); ok {
			req.Width = int(widthVal)
		}
	}
	if height, ok := params["height"]; ok {
		if heightVal, ok := height.(float64); ok {
			req.Height = int(heightVal)
		}
	}
	if scale, ok := params["scale"]; ok {
		if scaleVal, ok := scale.(float64); ok {
			req.Scale = scaleVal
		}
	}
	req.ForceSingle = true
}

// setJimengV4I2iParams 设置即梦4.0图生图参数
func (s *Service) setJimengV4I2iParams(req *SubmitTaskRequest, params map[string]any) {
	if imageUrls, ok := params["image_urls"].([]string); ok {
		req.ImageUrls = append(req.ImageUrls, imageUrls...)
	}
	if width, ok := params["width"]; ok {
		if widthVal, ok := width.(float64); ok {
			req.Width = int(widthVal)
		}
	}
	if height, ok := params["height"]; ok {
		if heightVal, ok := height.(float64); ok {
			req.Height = int(heightVal)
		}
	}
	if scale, ok := params["scale"]; ok {
		if scaleVal, ok := scale.(float64); ok {
			req.Scale = scaleVal
		}
	}
	req.ForceSingle = true
}

// setCommonParams 设置通用参数（seed, width, height等）
func (s *Service) setCommonParams(req *SubmitTaskRequest, params map[string]any) {
	if seed, ok := params["seed"]; ok {
		if seedVal, err := strconv.ParseInt(fmt.Sprintf("%.0f", seed), 10, 64); err == nil {
			req.Seed = seedVal
		}
	}
	if width, ok := params["width"]; ok {
		if widthVal, ok := width.(float64); ok {
			req.Width = int(widthVal)
		}
	}
	if height, ok := params["height"]; ok {
		if heightVal, ok := height.(float64); ok {
			req.Height = int(heightVal)
		}
	}
}

// pollTaskStatus 轮询任务状态
func (s *Service) pollTaskStatus() {
	for {
		var jobs []model.JimengJob
		s.db.Where("status IN (?)", []model.JMTaskStatus{model.JMTaskStatusGenerating, model.JMTaskStatusInQueue}).Find(&jobs)
		if len(jobs) == 0 {
			logger.Debugf("no jimeng task to poll, sleep 10s")
			time.Sleep(10 * time.Second)
			continue
		}

		for _, job := range jobs {
			if job.UpdatedAt.Before(time.Now().Add(-10 * time.Minute)) {
				s.handleTaskError(job.Id, "task timeout")
				continue
			}

			if job.Type == model.JMTaskTypeJimengV4T2i || job.Type == model.JMTaskTypeJimengV4I2i {
				s.pollV4Task(job)
			} else {
				s.pollSdkTask(job)
			}
		}

		time.Sleep(5 * time.Second)
	}
}

// pollSdkTask 轮询SDK任务状态
func (s *Service) pollSdkTask(job model.JimengJob) {
	resp, err := s.client.QueryTask(&QueryTaskRequest{
		ReqKey:  job.ReqKey,
		TaskId:  job.TaskId,
		ReqJson: `{"return_url":true}`,
	})
	if err != nil {
		s.handleTaskError(job.Id, fmt.Sprintf("query task failed: %s", err.Error()))
		return
	}

	rawData, _ := json.Marshal(resp)
	s.db.Model(&model.JimengJob{}).Where("id = ?", job.Id).Update("raw_data", string(rawData))

	if resp.Code != 10000 {
		s.handleTaskError(job.Id, fmt.Sprintf("query task failed: %s", resp.Message))
		return
	}

	switch resp.Data.Status {
	case model.JMTaskStatusDone:
		if resp.Message != "Success" {
			s.handleTaskError(job.Id, fmt.Sprintf("task failed: %s", resp.Data.AlgorithmBaseResp.StatusMessage))
			return
		}
		updates := map[string]any{
			"status":     model.JMTaskStatusSuccess,
			"updated_at": time.Now(),
		}
		if len(resp.Data.ImageUrls) > 0 {
			imgUrl, err := s.uploader.GetUploadHandler().PutUrlFile(resp.Data.ImageUrls[0], ".png", false)
			if err != nil {
				logger.Errorf("upload image failed: %v", err)
				imgUrl = resp.Data.ImageUrls[0]
			}
			updates["img_url"] = imgUrl
		}
		if resp.Data.VideoUrl != "" {
			videoUrl, err := s.uploader.GetUploadHandler().PutUrlFile(resp.Data.VideoUrl, ".mp4", false)
			if err != nil {
				logger.Errorf("upload video failed: %v", err)
				videoUrl = resp.Data.VideoUrl
			}
			updates["video_url"] = videoUrl
		}
		s.db.Model(&model.JimengJob{}).Where("id = ?", job.Id).Updates(updates)
	case model.JMTaskStatusInQueue, model.JMTaskStatusGenerating:
		s.UpdateJobStatus(job.Id, model.JMTaskStatusGenerating, "")
	case model.JMTaskStatusNotFound:
		s.handleTaskError(job.Id, "task not found")
	case model.JMTaskStatusExpired:
		// skip
	default:
		logger.Warnf("unknown task status: %s", resp.Data.Status)
	}
}

// pollV4Task 轮询即梦4.0任务状态
func (s *Service) pollV4Task(job model.JimengJob) {
	resp, err := s.client.QueryV4Task(&V4QueryRequest{
		ReqKey:  job.ReqKey,
		TaskId:  job.TaskId,
		ReqJson: `{"return_url":true}`,
	})
	if err != nil {
		s.handleTaskError(job.Id, fmt.Sprintf("query v4 task failed: %s", err.Error()))
		return
	}

	rawData, _ := json.Marshal(resp)
	s.db.Model(&model.JimengJob{}).Where("id = ?", job.Id).Update("raw_data", string(rawData))

	if resp.Code != 10000 {
		errMsg := mapV4ErrorMessage(resp.Code, resp.Message)
		s.handleTaskError(job.Id, fmt.Sprintf("query v4 task failed: %s", errMsg))
		return
	}

	switch resp.Data.Status {
	case model.JMTaskStatusDone:
		updates := map[string]any{
			"status":     model.JMTaskStatusSuccess,
			"updated_at": time.Now(),
		}
		if len(resp.Data.ImageUrls) > 0 {
			imgUrl, err := s.uploader.GetUploadHandler().PutUrlFile(resp.Data.ImageUrls[0], ".png", false)
			if err != nil {
				logger.Errorf("upload v4 image failed: %v", err)
				imgUrl = resp.Data.ImageUrls[0]
			}
			updates["img_url"] = imgUrl
		}
		s.db.Model(&model.JimengJob{}).Where("id = ?", job.Id).Updates(updates)
	case model.JMTaskStatusInQueue, model.JMTaskStatusGenerating:
		s.UpdateJobStatus(job.Id, model.JMTaskStatusGenerating, "")
	case model.JMTaskStatusNotFound:
		s.handleTaskError(job.Id, "task not found")
	case model.JMTaskStatusExpired:
		// skip
	default:
		logger.Warnf("unknown v4 task status: %s", resp.Data.Status)
	}
}

// mapV4ErrorMessage 映射即梦4.0错误码为中文错误信息
func mapV4ErrorMessage(code int, msg string) string {
	switch code {
	case 50411:
		return "输入图片审核未通过"
	case 50511:
		return "输出图片审核未通过，请重试"
	case 50412:
		return "输入文本审核未通过"
	case 50512:
		return "输出文本审核未通过"
	case 50413:
		return "输入文本包含敏感词或版权词"
	case 50518:
		return "输入版权图审核未通过"
	case 50519:
		return "输出版权图审核未通过，请重试"
	case 50520, 50521, 50522:
		return "审核服务异常"
	case 50429:
		return "请求过于频繁，请稍后重试"
	case 50430:
		return "并发请求超限，请稍后重试"
	case 50500, 50501:
		return "服务内部错误"
	default:
		return msg
	}
}

// UpdateJobStatus 更新任务状态
func (s *Service) UpdateJobStatus(jobId uint, status model.JMTaskStatus, errMsg string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errMsg != "" {
		updates["err_msg"] = errMsg
	}
	return s.db.Model(&model.JimengJob{}).Where("id = ?", jobId).Updates(updates).Error
}

// handleTaskError 处理任务错误
func (s *Service) handleTaskError(jobId uint, errMsg string) error {
	logger.Errorf("Jimeng task error (job_id: %d): %s", jobId, errMsg)
	return s.UpdateJobStatus(jobId, model.JMTaskStatusFailed, errMsg)
}

// PushTaskToQueue 推送任务到队列（用于手动重试）
func (s *Service) PushTaskToQueue(jobId uint) error {
	return s.taskQueue.RPush(jobId)
}

// GetTaskStats 获取任务统计信息
func (s *Service) GetTaskStats() (map[string]any, error) {
	type StatResult struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	var stats []StatResult
	err := s.db.Model(&model.JimengJob{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&stats).Error
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"total":      int64(0),
		"completed":  int64(0),
		"processing": int64(0),
		"failed":     int64(0),
		"pending":    int64(0),
	}

	for _, stat := range stats {
		result["total"] = result["total"].(int64) + stat.Count
		result[stat.Status] = stat.Count
	}

	return result, nil
}

// GetJob 获取任务
func (s *Service) GetJob(jobId uint) (*model.JimengJob, error) {
	var job model.JimengJob
	if err := s.db.First(&job, jobId).Error; err != nil {
		return nil, err
	}
	return &job, nil
}
