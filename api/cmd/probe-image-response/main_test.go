package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingEnvironmentValues(t *testing.T) {
	t.Setenv("API_KEY", "")
	t.Setenv("API_URL", "")
	t.Setenv("MODEL", "")
	t.Setenv("PROVIDER", "")

	path := filepath.Join(t.TempDir(), ".env")
	content := "\n# comment\nAPI_URL=https://example.test\nAPI_KEY='secret-key'\nMODEL=\"gpt-image-2\"\nPROVIDER=openai\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("API_URL"); got != "https://example.test" {
		t.Fatalf("API_URL = %q", got)
	}
	if got := os.Getenv("API_KEY"); got != "secret-key" {
		t.Fatalf("API_KEY = %q", got)
	}
	if got := os.Getenv("MODEL"); got != "gpt-image-2" {
		t.Fatalf("MODEL = %q", got)
	}
	if got := os.Getenv("PROVIDER"); got != "openai" {
		t.Fatalf("PROVIDER = %q", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnvironmentValues(t *testing.T) {
	t.Setenv("API_KEY", "from-env")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("API_KEY=from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("API_KEY"); got != "from-env" {
		t.Fatalf("API_KEY = %q", got)
	}
}
