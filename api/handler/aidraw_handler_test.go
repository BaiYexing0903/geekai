package handler

import (
	"testing"

	"geekai/core/types"
	"geekai/store/model"
	"geekai/store/vo"
	"geekai/utils"
)

func TestFillAiDrawJobModelNamesUsesChatModelDisplayName(t *testing.T) {
	jobs := []vo.AiDrawJob{
		{Id: 1, Prompt: "画一只猫"},
	}
	items := []model.AiDrawJob{
		{
			Id:       1,
			TaskInfo: utils.JsonEncode(types.AiDrawTask{ModelId: 7, ModelName: "gpt-image-2"}),
		},
	}
	models := []model.ChatModel{
		{Id: 7, Name: "GPT 图片模型", Value: "gpt-image-2"},
	}

	fillAiDrawJobModelNames(jobs, items, models)

	if jobs[0].ModelName != "GPT 图片模型" {
		t.Fatalf("expected display model name, got %q", jobs[0].ModelName)
	}
}
