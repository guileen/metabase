package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Config represents the client configuration
type Config struct {
	URL         string            `json:"url"`
	APIKey      string            `json:"apikey,omitempty"`
	AccessToken string            `json:"access_token,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	HTTPClient  *http.Client      `json:"-"`
	Database    *DatabaseConfig   `json:"db,omitempty"`
	Auth        *AuthConfig       `json:"auth,omitempty"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Schema  string        `json:"schema,omitempty"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	AutoRefreshToken bool    `json:"auto_refresh_token,omitempty"`
	PersistSession   bool    `json:"persist_session,omitempty"`
	Storage          Storage `json:"-"`
}

// Storage represents session storage interface
type Storage interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// MemoryStorage provides in-memory storage
type MemoryStorage struct {
	data map[string]string
	mu   sync.RWMutex
}

// NewMemoryStorage creates a new memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]string),
	}
}

// Get retrieves a value from memory storage
func (m *MemoryStorage) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key], nil
}

// Set stores a value in memory storage
func (m *MemoryStorage) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

// Delete removes a value from memory storage
func (m *MemoryStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// Record represents a database record
type Record struct {
	ID        string                 `json:"id"`
	Table     string                 `json:"table"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Version   int64                  `json:"version"`
}

// QueryOptions represents query options
type QueryOptions struct {
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	OrderBy  string                 `json:"order_by"`
	OrderDir string                 `json:"order_dir"`
	Where    map[string]interface{} `json:"where"`
	Select   []string               `json:"select"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	Records []Record               `json:"records"`
	Total   int                    `json:"total"`
	HasNext bool                   `json:"has_next"`
	Meta    map[string]interface{} `json:"meta"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error [%s]: %s", e.Code, e.Message)
}

// Client represents the MetaBase client
type Client struct {
	config *Config
	http   *http.Client
}

// New creates a new MetaBase client
func New(config *Config) *Client {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		config: config,
		http:   config.HTTPClient,
	}
}

// Create creates a new record
func (c *Client) Create(ctx context.Context, table string, data map[string]interface{}) (*Record, error) {
	request := map[string]interface{}{
		"table": table,
		"data":  data,
	}

	result, err := c.makeRequest(ctx, "POST", "/api/storage/create", request)
	if err != nil {
		return nil, err
	}

	var record Record
	if err := json.Unmarshal(result, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &record, nil
}

// Get retrieves a record by ID
func (c *Client) Get(ctx context.Context, table, id string) (*Record, error) {
	params := url.Values{}
	params.Set("table", table)
	params.Set("id", id)

	result, err := c.makeRequest(ctx, "GET", "/api/storage/get?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var record Record
	if err := json.Unmarshal(result, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &record, nil
}

// Update updates an existing record
func (c *Client) Update(ctx context.Context, table, id string, data map[string]interface{}) (*Record, error) {
	request := map[string]interface{}{
		"table": table,
		"id":    id,
		"data":  data,
	}

	result, err := c.makeRequest(ctx, "PUT", "/api/storage/update", request)
	if err != nil {
		return nil, err
	}

	var record Record
	if err := json.Unmarshal(result, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &record, nil
}

// Delete removes a record
func (c *Client) Delete(ctx context.Context, table, id string) error {
	params := url.Values{}
	params.Set("table", table)
	params.Set("id", id)

	_, err := c.makeRequest(ctx, "DELETE", "/api/storage/delete?"+params.Encode(), nil)
	return err
}

// Query performs a query with options
func (c *Client) Query(ctx context.Context, table string, options *QueryOptions) (*QueryResult, error) {
	if options == nil {
		options = &QueryOptions{Limit: 100}
	}

	params := url.Values{}
	params.Set("table", table)
	if options.Limit > 0 {
		params.Set("limit", strconv.Itoa(options.Limit))
	}
	if options.Offset > 0 {
		params.Set("offset", strconv.Itoa(options.Offset))
	}
	if options.OrderBy != "" {
		params.Set("order_by", options.OrderBy)
	}
	if options.OrderDir != "" {
		params.Set("order_dir", options.OrderDir)
	}

	// Add where conditions
	if options.Where != nil {
		for key, value := range options.Where {
			params.Set("where["+key+"]", fmt.Sprintf("%v", value))
		}
	}

	result, err := c.makeRequest(ctx, "GET", "/api/storage/query?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var queryResult QueryResult
	if err := json.Unmarshal(result, &queryResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &queryResult, nil
}

// UploadFile uploads a file
func (c *Client) UploadFile(ctx context.Context, file io.Reader, filename string, opts *UploadOptions) (*FileMetadata, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Add options as form fields
	if opts != nil {
		if opts.TenantID != "" {
			writer.WriteField("tenant_id", opts.TenantID)
		}
		if opts.ProjectID != "" {
			writer.WriteField("project_id", opts.ProjectID)
		}
		if opts.CreatedBy != "" {
			writer.WriteField("created_by", opts.CreatedBy)
		}
		if opts.IsPublic {
			writer.WriteField("is_public", "true")
		}
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.URL+"/api/files/upload", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.handleAPIError(resp)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var metadata FileMetadata
	if err := json.Unmarshal(result, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &metadata, nil
}

// TrackEvent tracks an analytics event
func (c *Client) TrackEvent(ctx context.Context, event *AnalyticsEvent) error {
	_, err := c.makeRequest(ctx, "POST", "/api/analytics/track", event)
	return err
}

// GetMetrics retrieves analytics metrics
func (c *Client) GetMetrics(ctx context.Context, metrics []Metric, filters *FilterOptions) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"metrics": metrics,
		"filters": filters,
	}

	result, err := c.makeRequest(ctx, "POST", "/api/analytics/metrics", request)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

// Health returns system health status
func (c *Client) Health(ctx context.Context) (map[string]interface{}, error) {
	result, err := c.makeRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health response: %w", err)
	}

	return response, nil
}

// makeRequest makes an HTTP request with authentication
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.config.URL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.setAuthHeader(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, c.handleAPIError(resp)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return result, nil
}

// setAuthHeader sets the authentication header
func (c *Client) setAuthHeader(req *http.Request) {
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	} else if c.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	}

	// Add custom headers
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}
}

// handleAPIError handles API error responses
func (c *Client) handleAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			Status:  resp.StatusCode,
			Message: "failed to read error response",
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &APIError{
			Status:  resp.StatusCode,
			Message: string(body),
		}
	}

	apiErr.Status = resp.StatusCode
	return &apiErr
}
