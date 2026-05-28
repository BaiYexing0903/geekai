package material

import (
	"path"
	"strings"
	"time"

	logger2 "geekai/logger"
	"geekai/store/model"

	"gorm.io/gorm"
)

var logger = logger2.GetLogger()

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RecordGenerated(userId uint, name string, fileURL string) {
	if userId == 0 || strings.TrimSpace(fileURL) == "" {
		return
	}

	ext := strings.ToLower(path.Ext(fileURL))
	if len(ext) > 10 {
		ext = ext[:10]
	}
	if name == "" {
		name = generatedFileName(ext)
	}
	if len(name) > 100 {
		name = name[:90] + ext
	}

	err := s.db.Create(&model.File{
		UserId:    userId,
		Name:      name,
		ObjKey:    fileURL,
		URL:       fileURL,
		Ext:       ext,
		Size:      0,
		CreatedAt: time.Now(),
	}).Error
	if err != nil {
		logger.Errorf("record generated material failed: %v", err)
	}
}

func generatedFileName(ext string) string {
	if ext == "" {
		return "generated-material"
	}
	return "generated-material" + ext
}
