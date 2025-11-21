package rag

import ("bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time")

// CloudClient 云模式客户端
type CloudClient struct {
	config *CloudConfig
	client *http.Client
}

// NewCloudClient 创建云客户端
func NewCloudClient(config *CloudConfig) (*CloudClient, error) {
	if config == nil {
		return nil, fmt.Errorf("云配置不能为空")
	}

	client := &CloudClient{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return client, nil
}

// CloudSearchRequest 云搜索请求
type CloudSearchRequest struct {
	Query           string   `json:"query"`
	TopK            int      `json:"top_k"`
	Window          int      `json:"window"`
	EnableExpansion bool     `json:"enable_expansion"`
	EnableSkills    bool     `json:"enable_skills"`
	IncludeGlobs    []string `json:"include_globs"`
	ExcludeGlobs    []string `json:"exclude_globs"`
}

// CloudSearchResponse 云搜索响应
type CloudSearchResponse struct {
	Results []*SearchResult `json:"results"`
	Stats   *CloudStats     `json:"stats"`
	Error   string          `json:"error,omitempty"`
}

// Search 执行云搜索
func (c *CloudClient) Search(ctx context.Context, req *CloudSearchRequest) ([]*SearchResult, error) {
	// 序列化请求
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/api/search", c.config.ServiceURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// 发送请求
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var searchResp CloudSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if searchResp.Error != "" {
		return nil, fmt.Errorf("搜索失败: %s", searchResp.Error)
	}

	return searchResp.Results, nil
}

// GetStats 获取云服务统计信息
func (c *CloudClient) GetStats() (*RAGStats, error) {
	url := fmt.Sprintf("%s/api/stats", c.config.ServiceURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建统计请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取统计信息失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取统计响应失败: %w", err)
	}

	var cloudStats CloudStats
	if err := json.Unmarshal(body, &cloudStats); err != nil {
		return nil, fmt.Errorf("解析统计信息失败: %w", err)
	}

	return &RAGStats{
		Mode:       CloudMode.String(),
		CloudStats: &cloudStats,
	}, nil
}

// Close 关闭云客户端
func (c *CloudClient) Close() error {
	// HTTP 客户端无需特殊关闭
	return nil
}

// Ping 检查云服务连接
func (c *CloudClient) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/ping", c.config.ServiceURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建 ping 请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping 失败，状态码: %d", resp.StatusCode)
	}

	return nil
}