package handler

import "testing"

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
