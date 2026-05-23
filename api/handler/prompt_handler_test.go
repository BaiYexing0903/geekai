package handler

import "testing"

func TestNormalizeGeneratedPromptRejectsEmptyContent(t *testing.T) {
	_, err := normalizeGeneratedPrompt("\"\"")
	if err == nil {
		t.Fatal("expected empty generated prompt to be rejected")
	}
}

func TestNormalizeGeneratedPromptTrimsWrappingQuotes(t *testing.T) {
	prompt, err := normalizeGeneratedPrompt("\"cinematic cat portrait\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt != "cinematic cat portrait" {
		t.Fatalf("expected trimmed prompt, got %q", prompt)
	}
}
