package seedance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"geekai/core/types"
	logger2 "geekai/logger"
)

var logger = logger2.GetLogger()

type Client struct {
	config     types.SeedanceConfig
	httpClient *http.Client
}

func NewClient(sysConfig *types.SystemConfig) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	c.UpdateConfig(sysConfig.Seedance)
	return c
}

func (c *Client) UpdateConfig(config types.SeedanceConfig) {
	c.config = config
}

func (c *Client) IsConfigured() bool {
	return c.config.ApiURL != "" && c.config.BearerToken != ""
}

func (c *Client) CreateTask(req *CreateTaskReq) (*CreateTaskResp, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	url := c.config.ApiURL + "/doubao/create"
	respBody, err := c.doPost(url, body)
	if err != nil {
		return nil, err
	}

	var result CreateTaskResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}

	logger.Infof("Seedance CreateTask response: %s", string(respBody))

	if result.Code != "" && result.Code != "200" {
		return nil, fmt.Errorf("API error: code=%s, message=%s", result.Code, result.Message)
	}

	return &result, nil
}

func (c *Client) QueryTask(taskId string) (*QueryTaskResp, error) {
	body, err := json.Marshal(map[string]string{"id": taskId})
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	url := c.config.ApiURL + "/doubao/get_result"
	respBody, err := c.doPost(url, body)
	if err != nil {
		return nil, err
	}

	var result QueryTaskResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}

	return &result, nil
}

func (c *Client) doPost(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
