package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type requestConfig struct {
	Provider string
	APIURL   string
	APIKey   string
	Model    string
	Prompt   string
	Size     string
	Quality  string
}

func main() {
	if envFile := env("ENV_FILE", ".env"); envFile != "" {
		if err := loadDotEnv(envFile); err != nil && !os.IsNotExist(err) {
			fatalf("load env file %s: %v", envFile, err)
		}
	}

	cfg := requestConfig{
		Provider: strings.ToLower(env("PROVIDER", "openai")),
		APIURL:   strings.TrimRight(env("API_URL", ""), "/"),
		APIKey:   env("API_KEY", ""),
		Model:    env("MODEL", ""),
		Prompt:   env("PROMPT", "draw a small red apple on a white background"),
		Size:     env("SIZE", "1024x1024"),
		Quality:  env("QUALITY", "medium"),
	}

	if cfg.APIURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		fatalf("missing required env: API_URL, API_KEY, MODEL")
	}

	var (
		method string
		url    string
		body   any
		headers = map[string]string{"Content-Type": "application/json"}
	)

	switch cfg.Provider {
	case "gemini", "banana", "nano-banana", "nano-banana-pro":
		method = http.MethodPost
		url = fmt.Sprintf("%s/v1beta/models/%s:generateContent", cfg.APIURL, cfg.Model)
		headers["x-goog-api-key"] = cfg.APIKey
		body = map[string]any{
			"contents": []any{
				map[string]any{
					"parts": []any{map[string]any{"text": cfg.Prompt}},
				},
			},
			"generationConfig": map[string]any{
				"responseModalities": []string{"TEXT", "IMAGE"},
			},
		}
	case "openai", "gpt-image", "gpt-image-2":
		method = http.MethodPost
		url = cfg.APIURL + "/v1/images/generations"
		headers["Authorization"] = "Bearer " + cfg.APIKey
		body = map[string]any{
			"model":   cfg.Model,
			"prompt":  cfg.Prompt,
			"n":       1,
			"size":    cfg.Size,
			"quality": cfg.Quality,
		}
	default:
		fatalf("unsupported PROVIDER %q, use openai or gemini", cfg.Provider)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		fatalf("marshal request: %v", err)
	}

	fmt.Printf("REQUEST %s %s\n", method, url)
	fmt.Printf("MODEL=%s PROVIDER=%s\n", cfg.Model, cfg.Provider)
	fmt.Printf("BODY %s\n\n", mustPretty(payload))

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		fatalf("create request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fatalf("read response: %v", err)
	}

	fmt.Printf("STATUS %s\n", resp.Status)
	fmt.Printf("CONTENT-TYPE %s\n", resp.Header.Get("Content-Type"))
	fmt.Println("FIELDS")
	printFields(respBody)
	fmt.Println("\nRAW JSON")
	fmt.Println(mustPretty(respBody))
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"'")
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func mustPretty(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(pretty)
}

func printFields(data []byte) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Printf("- non-json response: %v\n", err)
		return
	}
	walk("$", v)
}

func walk(path string, v any) {
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			fmt.Printf("- %s: empty object\n", path)
			return
		}
		for k, val := range x {
			walk(path+"."+k, val)
		}
	case []any:
		fmt.Printf("- %s: array len=%d\n", path, len(x))
		if len(x) > 0 {
			walk(path+"[0]", x[0])
		}
	case string:
		fmt.Printf("- %s: string len=%d preview=%q\n", path, len(x), preview(x))
	case nil:
		fmt.Printf("- %s: null\n", path)
	default:
		fmt.Printf("- %s: %T=%v\n", path, x, x)
	}
}

func preview(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) <= 80 {
		return s
	}
	return s[:80] + "..."
}
