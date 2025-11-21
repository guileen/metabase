package api

import ("context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time")

// Client represents API client
type Client struct {
	BaseURL      string
	APIKey       string
	Timeout      time.Duration
	RetryAttempts int
	RetryDelay   time.Duration
	client       *http.Client
}

// ClientConfig represents client configuration
type ClientConfig struct {
	BaseURL       string        `json:"base_url"`
	APIKey        string        `json:"api_key"`
	Timeout       time.Duration `json:"timeout"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`
}

// Response represents API response
type Response struct {
	StatusCode int
	Data       interface{}
	Headers    map[string][]string
}

// NewClient creates a new API client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		BaseURL:      config.BaseURL,
		APIKey:       config.APIKey,
		Timeout:      timeout,
		RetryAttempts: config.RetryAttempts,
		RetryDelay:   config.RetryDelay,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.doRequest(ctx, "GET", path, nil)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, data interface{}) (*Response, error) {
	return c.doRequest(ctx, "POST", path, data)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, data interface{}) (*Response, error) {
	return c.doRequest(ctx, "PUT", path, data)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.doRequest(ctx, "DELETE", path, nil)
}

// doRequest performs an HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, data interface{}) (*Response, error) {
	url := c.BaseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Perform request with retry
	var lastErr error
	for attempt := 0; attempt <= c.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.RetryDelay)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Parse response data
		var responseData interface{}
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &responseData); err != nil {
				// If JSON parsing fails, use raw string
				responseData = string(respBody)
			}
		}

		return &Response{
			StatusCode: resp.StatusCode,
			Data:       responseData,
			Headers:    resp.Header,
		}, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.RetryAttempts+1, lastErr)
}