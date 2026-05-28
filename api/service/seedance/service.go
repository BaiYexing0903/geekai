package seedance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"geekai/service/material"
	"geekai/service/oss"
	"geekai/store"
	"geekai/store/model"

	"github.com/go-redis/redis/v8"
)

type Service struct {
	db              *gorm.DB
	redis           *redis.Client
	taskQueue       *store.RedisQueue
	client          *Client
	ctx             context.Context
	cancel          context.CancelFunc
	running         bool
	uploader        *oss.UploaderManager
	materialService *material.Service
}

func NewService(db *gorm.DB, redisCli *redis.Client, uploader *oss.UploaderManager, client *Client, materialService *material.Service) *Service {
	taskQueue := store.NewRedisQueue("SeedanceTaskQueue", redisCli)
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:              db,
		redis:           redisCli,
		taskQueue:       taskQueue,
		client:          client,
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
		uploader:        uploader,
		materialService: materialService,
	}
}

func (s *Service) Start() {
	if s.running {
		return
	}
	logger.Info("Starting Seedance service and task consumer...")
	s.running = true
	go s.consumeTasks()
	go s.pollTaskStatus()
}

func (s *Service) Stop() {
	if !s.running {
		return
	}
	logger.Info("Stopping Seedance service...")
	s.running = false
	s.cancel()
}

func (s *Service) consumeTasks() {
	for {
		select {
		case <-s.ctx.Done():
			logger.Info("Seedance task consumer stopped")
			return
		default:
			s.processNextTask()
		}
	}
}

func (s *Service) processNextTask() {
	var jobId uint
	if err := s.taskQueue.LPop(&jobId); err != nil {
		time.Sleep(time.Second)
		return
	}

	logger.Infof("Processing Seedance task: job_id=%d", jobId)

	if err := s.ProcessTask(jobId); err != nil {
		logger.Errorf("process seedance task failed: job_id=%d, error=%v", jobId, err)
		s.UpdateJobStatus(jobId, model.SDStatusFailed, err.Error())
	}
}

func (s *Service) ListMediaAssetGroup(req *ListMediaAssetGroupReq) (*ListMediaAssetGroupResp, error) {
	return s.client.ListMediaAssetGroup(req)
}

func (s *Service) CreateAsset(req *CreateAssetReq) (*CreateAssetResp, error) {
	return s.client.CreateAsset(req)
}

func (s *Service) CreateTask(userId uint, req *CreateJobReq) (*model.SeedanceJob, error) {
	taskParams := map[string]any{
		"content":        req.Content,
		"generate_audio": req.GenerateAudio,
		"resolution":     req.Resolution,
		"ratio":          req.Ratio,
		"duration":       req.Duration,
		"watermark":      req.Watermark,
	}
	paramsJson, err := json.Marshal(taskParams)
	if err != nil {
		return nil, fmt.Errorf("marshal task params failed: %w", err)
	}

	job := &model.SeedanceJob{
		UserId:     userId,
		Type:       req.Mode,
		Model:      req.Model,
		Prompt:     req.Prompt,
		TaskParams: string(paramsJson),
		Status:     model.SDStatusQueued,
		Power:      req.Power,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("create seedance job failed: %w", err)
	}

	if err := s.taskQueue.RPush(job.Id); err != nil {
		return nil, fmt.Errorf("push seedance task to queue failed: %w", err)
	}

	return job, nil
}

func (s *Service) ProcessTask(jobId uint) error {
	var job model.SeedanceJob
	if err := s.db.First(&job, jobId).Error; err != nil {
		return fmt.Errorf("get seedance job failed: %w", err)
	}

	if err := s.UpdateJobStatus(job.Id, model.SDStatusRunning, ""); err != nil {
		return fmt.Errorf("update job status failed: %w", err)
	}

	// 解析任务参数
	var params map[string]any
	if err := json.Unmarshal([]byte(job.TaskParams), &params); err != nil {
		return s.handleTaskError(job.Id, fmt.Sprintf("parse task params failed: %v", err))
	}

	// 构建 content 数组
	contentJson, _ := json.Marshal(params["content"])
	var content []ContentItem
	if err := json.Unmarshal(contentJson, &content); err != nil {
		return s.handleTaskError(job.Id, fmt.Sprintf("parse content failed: %v", err))
	}

	// 构建请求
	req := &CreateTaskReq{
		Model:   job.Model,
		Content: content,
	}
	if v, ok := params["generate_audio"].(bool); ok {
		req.GenerateAudio = v
	}
	if v, ok := params["resolution"].(string); ok {
		req.Resolution = v
	}
	if v, ok := params["ratio"].(string); ok {
		req.Ratio = v
	}
	if v, ok := params["duration"].(float64); ok {
		req.Duration = int(v)
	}
	if v, ok := params["watermark"].(bool); ok {
		req.Watermark = v
	}

	resp, err := s.client.CreateTask(req)
	if err != nil {
		return s.handleTaskError(job.Id, fmt.Sprintf("create task failed: %v", err))
	}

	if resp.ID == "" {
		return s.handleTaskError(job.Id, fmt.Sprintf("create task returned empty id: %s", resp.Message))
	}

	// 更新 API 任务 ID
	rawData, _ := json.Marshal(resp)
	if err := s.db.Model(&model.SeedanceJob{}).Where("id = ?", job.Id).Updates(map[string]any{
		"task_id":    resp.ID,
		"raw_data":   string(rawData),
		"updated_at": time.Now(),
	}).Error; err != nil {
		logger.Errorf("update seedance job task_id failed: %v", err)
	}

	return nil
}

func (s *Service) pollTaskStatus() {
	for {
		select {
		case <-s.ctx.Done():
			logger.Info("Seedance poll stopped")
			return
		default:
		}

		var jobs []model.SeedanceJob
		s.db.Where("status IN (?)", []model.SDTaskStatus{model.SDStatusQueued, model.SDStatusRunning}).Find(&jobs)
		if len(jobs) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, job := range jobs {
			// 超时处理（15分钟）
			if job.UpdatedAt.Before(time.Now().Add(-15 * time.Minute)) {
				s.handleTaskError(job.Id, "task timeout")
				continue
			}

			if job.TaskId == "" {
				continue
			}

			resp, err := s.client.QueryTask(job.TaskId)
			if err != nil {
				logger.Errorf("query seedance task failed: job_id=%d, error=%v", job.Id, err)
				continue
			}

			// 更新原始数据
			rawData, _ := json.Marshal(resp)
			s.db.Model(&model.SeedanceJob{}).Where("id = ?", job.Id).Update("raw_data", string(rawData))

			switch resp.Status {
			case string(model.SDStatusSucceeded):
				updates := map[string]any{
					"status":     model.SDStatusSucceeded,
					"updated_at": time.Now(),
				}

				// 下载视频到 OSS
				if resp.Content != nil && resp.Content.VideoURL != "" {
					videoUrl, err := s.uploader.GetUploadHandler().PutUrlFile(resp.Content.VideoURL, ".mp4", false)
					if err != nil {
						logger.Errorf("upload seedance video failed: %v", err)
						videoUrl = resp.Content.VideoURL
					}
					updates["video_url"] = videoUrl
					s.materialService.RecordGenerated(job.UserId, "seedance.mp4", videoUrl)
				}

				// 下载末帧图
				if resp.Content != nil && resp.Content.LastFrameURL != "" {
					lastFrameUrl, err := s.uploader.GetUploadHandler().PutUrlFile(resp.Content.LastFrameURL, ".png", false)
					if err != nil {
						logger.Errorf("upload seedance last frame failed: %v", err)
						lastFrameUrl = resp.Content.LastFrameURL
					}
					updates["last_frame_url"] = lastFrameUrl
				}

				s.db.Model(&model.SeedanceJob{}).Where("id = ?", job.Id).Updates(updates)

			case string(model.SDStatusQueued), string(model.SDStatusRunning):
				s.UpdateJobStatus(job.Id, model.SDTaskStatus(resp.Status), "")

			case string(model.SDStatusFailed):
				errMsg := "task failed"
				if resp.Error != nil {
					errMsg = resp.Error.Message
				}
				s.handleTaskError(job.Id, errMsg)
				// 退回算力
				if job.Power > 0 {
					s.refundPower(job.UserId, job.Power, job.Id)
				}

			case string(model.SDStatusExpired):
				s.handleTaskError(job.Id, "task expired")
				if job.Power > 0 {
					s.refundPower(job.UserId, job.Power, job.Id)
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func (s *Service) UpdateJobStatus(jobId uint, status model.SDTaskStatus, errMsg string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errMsg != "" {
		updates["err_msg"] = errMsg
	}
	return s.db.Model(&model.SeedanceJob{}).Where("id = ?", jobId).Updates(updates).Error
}

func (s *Service) handleTaskError(jobId uint, errMsg string) error {
	logger.Errorf("Seedance task error (job_id: %d): %s", jobId, errMsg)
	return s.UpdateJobStatus(jobId, model.SDStatusFailed, errMsg)
}

func (s *Service) refundPower(userId uint, power int, jobId uint) {
	// 通过 userService 退回算力（在 handler 层处理更合适，这里简化处理）
	// 实际退回在 Remove 接口中处理
}

func (s *Service) PushTaskToQueue(jobId uint) error {
	return s.taskQueue.RPush(jobId)
}

func (s *Service) GetJob(jobId uint) (*model.SeedanceJob, error) {
	var job model.SeedanceJob
	if err := s.db.First(&job, jobId).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Service) GetStats() (map[string]any, error) {
	type StatResult struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	var stats []StatResult
	err := s.db.Model(&model.SeedanceJob{}).
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
		switch stat.Status {
		case string(model.SDStatusQueued):
			result["pending"] = stat.Count
		case string(model.SDStatusRunning):
			result["processing"] = stat.Count
		case string(model.SDStatusSucceeded):
			result["completed"] = stat.Count
		case string(model.SDStatusFailed):
			result["failed"] = stat.Count
		}
	}

	return result, nil
}
