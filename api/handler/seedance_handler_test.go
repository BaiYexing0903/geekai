package handler

import (
	"testing"

	"geekai/store/model"
)

func TestSeedanceTaskTypeAllowsOnlyMultimodalReference(t *testing.T) {
	allowedTypes := []string{"multimodal_ref"}
	for _, taskType := range allowedTypes {
		if !isSeedanceTaskTypeAllowed(taskType) {
			t.Fatalf("expected %s to be allowed", taskType)
		}
	}

	blockedTypes := []string{
		"text_to_video",
		"image_to_video_first",
		"image_to_video_dual",
		"edit_video",
		"extend_video",
		"virtual_avatar",
		"",
	}
	for _, taskType := range blockedTypes {
		if isSeedanceTaskTypeAllowed(taskType) {
			t.Fatalf("expected %s to be blocked", taskType)
		}
	}
}

func TestSeedanceStatusFilterMapsUiFiltersToJobStatuses(t *testing.T) {
	tests := []struct {
		filter string
		want   []model.SDTaskStatus
	}{
		{filter: "processing", want: []model.SDTaskStatus{model.SDStatusQueued, model.SDStatusRunning}},
		{filter: "succeeded", want: []model.SDTaskStatus{model.SDStatusSucceeded}},
		{filter: "failed", want: []model.SDTaskStatus{model.SDStatusFailed, model.SDStatusExpired}},
		{filter: "all", want: nil},
		{filter: "", want: nil},
	}

	for _, tt := range tests {
		got := seedanceStatusFilter(tt.filter)
		if len(got) != len(tt.want) {
			t.Fatalf("filter %q expected %v, got %v", tt.filter, tt.want, got)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("filter %q expected %v, got %v", tt.filter, tt.want, got)
			}
		}
	}
}
