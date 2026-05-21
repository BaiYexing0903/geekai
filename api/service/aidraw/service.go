package aidraw

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"geekai/core/types"
	logger2 "geekai/logger"
	"geekai/service"
	"geekai/service/oss"
	"geekai/store"
	"geekai/store/model"
	"geekai/utils"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/imroc/req/v3"
	"gorm.io/gorm"
)

var logger = logger2.GetLogger()

type Service struct {
	httpClient    *req.Client
	db            *gorm.DB
	uploadManager *oss.UploaderManager
	taskQueue     *store.RedisQueue
	userService   *service.UserService
}

func NewService(db *gorm.DB, manager *oss.UploaderManager, redisCli *redis.Client, userService *service.UserService) *Service {
	return &Service{
		httpClient:    req.C().SetTimeout(time.Minute * 5),
		db:            db,
		taskQueue:     store.NewRedisQueue("AiDraw_Task_Queue", redisCli),
		uploadManager: manager,
		userService:   userService,
	}
}

func (s *Service) PushTask(task types.AiDrawTask) {
	logger.Infof("add a new AiDraw task to the task list: %+v", task)
	if err := s.taskQueue.RPush(task); err != nil {
		logger.Errorf("push aidraw task to queue failed: %v", err)
	}
}

func (s *Service) Run() {
	var jobs []model.AiDrawJob
	s.db.Where("progress", 0).Find(&jobs)
	for _, v := range jobs {
		var task types.AiDrawTask
		err := utils.JsonDecode(v.TaskInfo, &task)
		if err != nil {
			logger.Errorf("decode task info with error: %v", err)
			continue
		}
		task.Id = v.Id
		s.PushTask(task)
	}

	logger.Info("Starting AiDraw job consumer...")
	go func() {
		for {
			var task types.AiDrawTask
			err := s.taskQueue.LPop(&task)
			if err != nil {
				logger.Errorf("taking task with error: %v", err)
				continue
			}
			logger.Infof("handle a new AiDraw task: %+v", task)
			go func() {
				_, err = s.Image(task)
				if err != nil {
					logger.Errorf("error with aidraw image task: %v", err)
					s.db.Model(&model.AiDrawJob{Id: task.Id}).UpdateColumns(map[string]interface{}{
						"progress": service.FailTaskProgress,
						"err_msg":  err.Error(),
					})
				}
			}()
		}
	}()
}

// ---- Gemini types ----

type geminiReq struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGenConfig  `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string             `json:"text,omitempty"`
	InlineData *geminiInlineData  `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiGenConfig struct {
	ResponseModalities []string        `json:"responseModalities"`
	ImageConfig        *geminiImageCfg `json:"imageConfig,omitempty"`
}

type geminiImageCfg struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

type geminiRes struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData *geminiInlineData `json:"inlineData,omitempty"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// ---- GPT-Image-2 types ----

type gptImgReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	N      int    `json:"n,omitempty"`
}

type gptImgRes struct {
	Data []struct {
		B64Json string `json:"b64_json,omitempty"`
	} `json:"data"`
}

type errRes struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Service) Image(task types.AiDrawTask) (string, error) {
	var chatModel model.ChatModel
	if task.ModelId > 0 {
		s.db.Where("id", task.ModelId).First(&chatModel)
	} else {
		s.db.Where("value", task.ModelName).First(&chatModel)
	}

	var apiKey model.ApiKey
	session := s.db.Where("enabled", true)
	if chatModel.KeyId > 0 {
		session = session.Where("id = ?", chatModel.KeyId)
	} else {
		session = session.Where("type = ?", "aidraw")
	}
	err := session.Order("last_used_at ASC").First(&apiKey).Error
	if err != nil {
		return "", fmt.Errorf("no available AiDraw api key: %v", err)
	}

	if len(apiKey.ProxyURL) > 5 {
		s.httpClient.SetProxyURL(apiKey.ProxyURL)
	}

	var imgURL string
	if strings.Contains(chatModel.Value, "gemini") {
		imgURL, err = s.callGemini(task, apiKey, chatModel)
	} else {
		imgURL, err = s.callGPTImage(task, apiKey, chatModel)
	}

	if err != nil {
		return "", err
	}

	// clean base64 data: remove data URI prefix, fix URL-safe chars, strip whitespace
	imgURL = cleanBase64(imgURL)

	// update api key last use time
	s.db.Model(&apiKey).UpdateColumn("last_used_at", time.Now().Unix())

	// upload to oss
	ossURL, err := s.uploadManager.GetUploadHandler().PutBase64(imgURL)
	if err != nil {
		return "", fmt.Errorf("error with upload image: %v", err)
	}

	err = s.db.Model(&model.AiDrawJob{Id: task.Id}).UpdateColumns(map[string]interface{}{
		"progress": 100,
		"prompt":   task.Prompt,
		"img_url":  ossURL,
		"org_url":  ossURL,
		"publish":  1,
	}).Error
	if err != nil {
		return "", fmt.Errorf("err with update database: %v", err)
	}

	return "", nil
}

func (s *Service) callGemini(task types.AiDrawTask, apiKey model.ApiKey, chatModel model.ChatModel) (string, error) {
	apiURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent", apiKey.ApiURL, chatModel.Value)

	parts := []geminiPart{{Text: task.Prompt}}

	// img2img: download images and add as inlineData parts
	if task.Mode == "image_to_image" && len(task.Images) > 0 {
		for _, imgURL := range task.Images {
			imgData, mimeType, err := s.downloadImageAsBase64(imgURL)
			if err != nil {
				logger.Errorf("download image for gemini failed: %v", err)
				continue
			}
			parts = append(parts, geminiPart{
				InlineData: &geminiInlineData{
					MimeType: mimeType,
					Data:     imgData,
				},
			})
		}
	}

	reqBody := geminiReq{
		Contents: []geminiContent{{Parts: parts}},
		GenerationConfig: geminiGenConfig{
			ResponseModalities: []string{"image"},
		},
	}

	if task.AspectRatio != "" || task.ImageSize != "" {
		reqBody.GenerationConfig.ImageConfig = &geminiImageCfg{
			AspectRatio: task.AspectRatio,
			ImageSize:   task.ImageSize,
		}
	}

	var res geminiRes
	var errResp errRes
	r, err := s.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("x-goog-api-key", apiKey.Value).
		SetBody(reqBody).
		SetErrorResult(&errResp).
		SetSuccessResult(&res).
		Post(apiURL)
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %v", err)
	}
	if r.IsErrorState() {
		return "", fmt.Errorf("gemini API error: %s", errResp.Error.Message)
	}

	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}

	for _, part := range res.Candidates[0].Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			return part.InlineData.Data, nil
		}
	}

	return "", fmt.Errorf("gemini response missing image data")
}

func (s *Service) callGPTImage(task types.AiDrawTask, apiKey model.ApiKey, chatModel model.ChatModel) (string, error) {
	if task.Mode == "image_to_image" && len(task.Images) > 0 {
		return s.callGPTImageEdit(task, apiKey, chatModel)
	}
	return s.callGPTImageGen(task, apiKey, chatModel)
}

func (s *Service) callGPTImageGen(task types.AiDrawTask, apiKey model.ApiKey, chatModel model.ChatModel) (string, error) {
	apiURL := fmt.Sprintf("%s/v1/images/generations", apiKey.ApiURL)

	reqBody := gptImgReq{
		Model:   chatModel.Value,
		Prompt:  task.Prompt,
		Size:    task.Size,
		Quality: task.Quality,
		N:       1,
	}

	var res gptImgRes
	var errResp errRes
	r, err := s.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetBody(reqBody).
		SetErrorResult(&errResp).
		SetSuccessResult(&res).
		Post(apiURL)
	if err != nil {
		return "", fmt.Errorf("gpt-image request failed: %v", err)
	}
	if r.IsErrorState() {
		return "", fmt.Errorf("gpt-image API error: %s", errResp.Error.Message)
	}

	if len(res.Data) == 0 || res.Data[0].B64Json == "" {
		return "", fmt.Errorf("gpt-image returned empty response")
	}

	return res.Data[0].B64Json, nil
}

func (s *Service) callGPTImageEdit(task types.AiDrawTask, apiKey model.ApiKey, chatModel model.ChatModel) (string, error) {
	apiURL := fmt.Sprintf("%s/v1/images/edits", apiKey.ApiURL)

	// build multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("model", chatModel.Value)
	_ = writer.WriteField("prompt", task.Prompt)
	_ = writer.WriteField("n", "1")
	if task.Size != "" {
		_ = writer.WriteField("size", task.Size)
	}
	if task.Quality != "" {
		_ = writer.WriteField("quality", task.Quality)
	}

	// add images
	for i, imgURL := range task.Images {
		imgData, mimeType, err := s.downloadImageAsBase64(imgURL)
		if err != nil {
			logger.Errorf("download image for gpt-edit failed: %v", err)
			continue
		}
		imgBytes, err := base64.StdEncoding.DecodeString(imgData)
		if err != nil {
			continue
		}
		ext := ".png"
		if mimeType == "image/jpeg" || mimeType == "image/jpg" {
			ext = ".jpg"
		} else if mimeType == "image/webp" {
			ext = ".webp"
		}
		part, err := writer.CreateFormFile("image", fmt.Sprintf("input_%d%s", i, ext))
		if err != nil {
			continue
		}
		part.Write(imgBytes)
	}

	writer.Close()

	// use raw http client for multipart
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("create gpt-edit request failed: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey.Value)

	client := &http.Client{Timeout: time.Minute * 5}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gpt-edit request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read gpt-edit response failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errRes
		json.Unmarshal(respBody, &errResp)
		return "", fmt.Errorf("gpt-edit API error (%d): %s", resp.StatusCode, errResp.Error.Message)
	}

	var res gptImgRes
	err = json.Unmarshal(respBody, &res)
	if err != nil {
		return "", fmt.Errorf("decode gpt-edit response failed: %v", err)
	}

	if len(res.Data) == 0 || res.Data[0].B64Json == "" {
		return "", fmt.Errorf("gpt-edit returned empty response")
	}

	return res.Data[0].B64Json, nil
}

func (s *Service) CheckTaskStatus() {
	go func() {
		logger.Info("Running AiDraw task status checking ...")
		for {
			var jobs []model.AiDrawJob
			s.db.Where("progress < ?", 100).Find(&jobs)
			for _, job := range jobs {
				if time.Since(job.CreatedAt) > time.Minute*10 {
					job.Progress = service.FailTaskProgress
					job.ErrMsg = "任务超时"
					s.db.Updates(&job)
				}
			}

			// refund power for failed tasks
			s.db.Where("progress", service.FailTaskProgress).Where("power > ?", 0).Find(&jobs)
			for _, job := range jobs {
				var task types.AiDrawTask
				err := utils.JsonDecode(job.TaskInfo, &task)
				if err != nil {
					continue
				}
				err = s.userService.IncreasePower(job.UserId, job.Power, model.PowerLog{
					Type:   types.PowerRefund,
					Model:  task.ModelName,
					Remark: fmt.Sprintf("AI绘画任务失败，退回算力。任务ID：%d，Err: %s", job.Id, job.ErrMsg),
				})
				if err != nil {
					continue
				}
				s.db.Model(&job).UpdateColumn("power", 0)
			}
			time.Sleep(time.Second * 10)
		}
	}()
}

func (s *Service) DownloadImages() {
	go func() {
		var items []model.AiDrawJob
		for {
			res := s.db.Where("img_url = ? AND progress = ?", "", 100).Find(&items)
			if res.Error != nil {
				continue
			}

			for _, v := range items {
				if v.OrgURL == "" {
					continue
				}
				logger.Infof("try to download aidraw image: %s", v.OrgURL)
				imgURL, err := s.uploadManager.GetUploadHandler().PutUrlFile(v.OrgURL, ".png", false)
				if err != nil {
					logger.Error("error with download aidraw image: %s, error: %v", v.OrgURL, err)
					continue
				}
				s.db.Model(&model.AiDrawJob{Id: v.Id}).UpdateColumn("img_url", imgURL)
				logger.Infof("download aidraw image %s successfully.", v.OrgURL)
			}

			time.Sleep(time.Second * 5)
		}
	}()
}

func (s *Service) downloadImageAsBase64(imgURL string) (string, string, error) {
	imageBytes, err := utils.DownloadImage(imgURL, "")
	if err != nil {
		return "", "", err
	}

	ext := strings.ToLower(filepath.Ext(imgURL))
	mimeType := "image/png"
	if mt := mime.TypeByExtension(ext); mt != "" {
		mimeType = mt
	}

	return base64.StdEncoding.EncodeToString(imageBytes), mimeType, nil
}

func cleanBase64(data string) string {
	// remove data URI prefix like "data:image/png;base64,"
	if idx := strings.Index(data, ","); idx >= 0 && strings.HasPrefix(data, "data:") {
		data = data[idx+1:]
	}
	// fix URL-safe base64: replace - with + and _ with /
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")
	// strip whitespace
	data = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, data)
	// fix padding
	if m := len(data) % 4; m > 0 {
		data += strings.Repeat("=", 4-m)
	}
	return data
}
