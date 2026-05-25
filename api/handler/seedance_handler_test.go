package handler

import (
	"testing"

	"geekai/service/seedance"
	"geekai/store/model"
)

func TestBuildSeedancePortraitFiltersAlwaysIncludesPortraitType(t *testing.T) {
	filters := buildSeedancePortraitFilters(SeedancePortraitListRequest{})
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(filters))
	}
	if filters[0].Field != "metadata.type" || filters[0].Op != "must" {
		t.Fatalf("unexpected type filter: %+v", filters[0])
	}
	if len(filters[0].Conds.StrValues) != 1 || filters[0].Conds.StrValues[0] != "portrait" {
		t.Fatalf("expected portrait filter, got %+v", filters[0].Conds.StrValues)
	}
}

func TestBuildSeedancePortraitFiltersAddsOptionalFilters(t *testing.T) {
	filters := buildSeedancePortraitFilters(SeedancePortraitListRequest{
		Gender:     "女性",
		Country:    "中国",
		Ages:       []string{"25", "26"},
		Occupation: "演员",
	})
	want := map[string][]string{
		"metadata.type":       {"portrait"},
		"metadata.gender":     {"女性"},
		"metadata.country":    {"中国"},
		"metadata.age":        {"25", "26"},
		"metadata.occupation": {"演员"},
	}
	if len(filters) != len(want) {
		t.Fatalf("expected %d filters, got %d", len(want), len(filters))
	}
	for _, filter := range filters {
		values, ok := want[filter.Field]
		if !ok {
			t.Fatalf("unexpected filter field %q", filter.Field)
		}
		if filter.Op != "must" {
			t.Fatalf("expected must op for %q", filter.Field)
		}
		if len(filter.Conds.StrValues) != len(values) {
			t.Fatalf("field %q expected %v, got %v", filter.Field, values, filter.Conds.StrValues)
		}
		for i := range values {
			if filter.Conds.StrValues[i] != values[i] {
				t.Fatalf("field %q expected %v, got %v", filter.Field, values, filter.Conds.StrValues)
			}
		}
	}
}

func TestNormalizeSeedancePortraitsExtractsImageAsset(t *testing.T) {
	resp := &seedance.ListMediaAssetGroupResp{Result: seedance.MediaAssetGroupResult{
		TotalCount: 1,
		PageNum:    1,
		PageSize:   24,
		Items: []seedance.MediaAssetGroupItem{{AssetGroup: seedance.MediaAssetGroup{
			Title:       "中国 22岁 女性 演员",
			Description: "22岁中国女性",
			Metadata:    seedance.MediaAssetMetadata{Country: "中国", Age: 22, Gender: "女性", Occupation: "演员", Type: "portrait"},
			Content:     seedance.MediaAssetContent{Image: []seedance.MediaAssetImage{{AssetID: "asset-abc", URL: "https://example.com/a.jpg"}}},
		}}},
	}}
	got := normalizeSeedancePortraits(resp)
	if got.Total != 1 || got.Page != 1 || got.PageSize != 24 || len(got.Items) != 1 {
		t.Fatalf("unexpected page result: %+v", got)
	}
	item := got.Items[0]
	if item.AssetID != "asset-abc" || item.AssetURL != "asset://asset-abc" || item.PreviewURL != "https://example.com/a.jpg" {
		t.Fatalf("unexpected portrait item: %+v", item)
	}
}

func TestNormalizeSeedancePortraitsSkipsItemsWithoutImageAsset(t *testing.T) {
	resp := &seedance.ListMediaAssetGroupResp{Result: seedance.MediaAssetGroupResult{Items: []seedance.MediaAssetGroupItem{{AssetGroup: seedance.MediaAssetGroup{Title: "missing"}}}}}
	got := normalizeSeedancePortraits(resp)
	if len(got.Items) != 0 {
		t.Fatalf("expected empty items, got %+v", got.Items)
	}
}

func TestSeedanceTaskTypeAllowsMultimodalReferenceAndDualFrameOnly(t *testing.T) {
	allowedTypes := []string{"multimodal_ref", "image_to_video_dual"}
	for _, taskType := range allowedTypes {
		if !isSeedanceTaskTypeAllowed(taskType) {
			t.Fatalf("expected %s to be allowed", taskType)
		}
	}

	blockedTypes := []string{
		"text_to_video",
		"image_to_video_first",
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

func TestNormalizeSeedanceCreatedAsset(t *testing.T) {
	got := normalizeSeedanceCreatedAsset(SeedanceCreateAssetRequest{
		URL:  "https://example.com/person.jpg",
		Name: "测试人像",
	}, &seedance.CreateAssetResp{ID: "asset-abc"})

	if got.ID != "asset-abc" {
		t.Fatalf("expected id asset-abc, got %q", got.ID)
	}
	if got.AssetURL != "asset://asset-abc" {
		t.Fatalf("expected asset url, got %q", got.AssetURL)
	}
	if got.PreviewURL != "https://example.com/person.jpg" {
		t.Fatalf("expected preview url, got %q", got.PreviewURL)
	}
	if got.Name != "测试人像" {
		t.Fatalf("expected name, got %q", got.Name)
	}
}
