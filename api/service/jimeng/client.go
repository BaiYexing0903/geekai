package jimeng

import (
	"bytes"
	"encoding/json"
	"fmt"
	"geekai/core/types"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/visual"
)

// Client 即梦API客户端
type Client struct {
	visual *visual.Visual
	config types.JimengConfig
}

// NewClient 创建即梦API客户端
func NewClient(sysConfig *types.SystemConfig) *Client {

	client := &Client{}
	client.UpdateConfig(sysConfig.Jimeng)
	return client
}

func (c *Client) UpdateConfig(config types.JimengConfig) error {
	// 使用官方SDK的visual实例
	visualInstance := visual.NewInstance()
	visualInstance.Client.SetAccessKey(config.AccessKey)
	visualInstance.Client.SetSecretKey(config.SecretKey)

	// 添加即梦AI专有的API配置
	jimengApis := map[string]*base.ApiInfo{
		"CVSync2AsyncSubmitTask": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"CVSync2AsyncSubmitTask"},
				"Version": []string{"2022-08-31"},
			},
		},
		"CVSync2AsyncGetResult": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"CVSync2AsyncGetResult"},
				"Version": []string{"2022-08-31"},
			},
		},
		"CVProcess": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"CVProcess"},
				"Version": []string{"2022-08-31"},
			},
		},
	}

	// 将即梦API添加到现有的ApiInfoList中
	for name, info := range jimengApis {
		visualInstance.Client.ApiInfoList[name] = info
	}

	c.config = config
	c.visual = visualInstance

	return c.testConnection()
}

// testConnection 测试即梦AI连接
func (c *Client) testConnection() error {

	// 使用一个简单的查询任务来测试连接
	testReq := &QueryTaskRequest{
		ReqKey: "test_connection",
		TaskId: "test_task_id_12345",
	}

	_, err := c.QueryTask(testReq)
	// 即使任务不存在，只要不是认证错误就说明连接正常
	if err != nil {
		// 检查是否是认证错误
		if strings.Contains(err.Error(), "InvalidAccessKey") {
			return fmt.Errorf("认证失败，请检查AccessKey和SecretKey是否正确")
		}
		// 其他错误（如任务不存在）说明连接正常
		return nil
	}
	return nil
}

// SubmitTask 提交异步任务
func (c *Client) SubmitTask(req *SubmitTaskRequest) (*SubmitTaskResponse, error) {
	// 直接将请求转为map[string]interface{}
	reqBodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 直接使用序列化后的字节
	jsonBody := reqBodyBytes

	// 调用SDK的JSON方法
	respBody, statusCode, err := c.visual.Client.Json("CVSync2AsyncSubmitTask", nil, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("submit task failed (status: %d): %w", statusCode, err)
	}

	logger.Infof("Jimeng SubmitTask Response: %s", string(respBody))

	// 解析响应
	var result SubmitTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// QueryTask 查询任务结果
func (c *Client) QueryTask(req *QueryTaskRequest) (*QueryTaskResponse, error) {
	// 序列化请求
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 调用SDK的JSON方法
	respBody, statusCode, err := c.visual.Client.Json("CVSync2AsyncGetResult", nil, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("query task failed (status: %d): %w", statusCode, err)
	}

	logger.Infof("Jimeng QueryTask Response: %s", string(respBody))

	// 解析响应
	var result QueryTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// SubmitSyncTask 提交同步任务（仅用于文生图）
func (c *Client) SubmitSyncTask(req *SubmitTaskRequest) (*QueryTaskResponse, error) {
	// 序列化请求
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	// 调用SDK的JSON方法
	respBody, statusCode, err := c.visual.Client.Json("CVProcess", nil, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("submit sync task failed (status: %d): %w", statusCode, err)
	}

	logger.Infof("Jimeng SubmitSyncTask Response: %s", string(respBody))

	// 解析响应，同步任务直接返回结果
	var result QueryTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// SubmitV4Task 提交即梦4.0异步任务（REST API）
func (c *Client) SubmitV4Task(req *V4CreateRequest) (*V4CreateResponse, error) {
	if c.config.ApiUrl == "" || c.config.BearerToken == "" {
		return nil, fmt.Errorf("即梦4.0 API未配置，请在后台设置API地址和Bearer Token")
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	apiUrl := strings.TrimRight(c.config.ApiUrl, "/") + "/api/v1/dream/create"
	httpReq, err := http.NewRequest(http.MethodPost, apiUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.BearerToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("submit v4 task failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	logger.Infof("Jimeng V4 SubmitTask Response: %s", string(respBody))

	var result V4CreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// QueryV4Task 查询即梦4.0任务结果（REST API）
func (c *Client) QueryV4Task(req *V4QueryRequest) (*V4QueryResponse, error) {
	if c.config.ApiUrl == "" || c.config.BearerToken == "" {
		return nil, fmt.Errorf("即梦4.0 API未配置")
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	apiUrl := strings.TrimRight(c.config.ApiUrl, "/") + "/api/v1/dream/get_result"
	httpReq, err := http.NewRequest(http.MethodPost, apiUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.BearerToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("query v4 task failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	logger.Infof("Jimeng V4 QueryTask Response: %s", string(respBody))

	var result V4QueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}
