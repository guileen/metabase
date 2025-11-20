package analytics

import ("context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/guileen/metabase/pkg/infra/storage")

// EventType represents different types of analytics events
type EventType string

const (
	EventTypePageView    EventType = "page_view"
	EventTypeAPIRequest  EventType = "api_request"
	EventTypeUserAction  EventType = "user_action"
	EventTypeError       EventType = "error"
	EventTypePerformance EventType = "performance"
	EventTypeConversion  EventType = "conversion"
	EventTypeCustom      EventType = "custom"
)

// Event represents an analytics event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType             `json:"type"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	EventType   string                 `json:"event_type"`
	EventName   string                 `json:"event_name"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    *int64                 `json:"duration,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	IP          string                 `json:"ip,omitempty"`
	Country     string                 `json:"country,omitempty"`
	City        string                 `json:"city,omitempty"`
	Device      string                 `json:"device,omitempty"`
	Browser     string                 `json:"browser,omitempty"`
	OS          string                 `json:"os,omitempty"`
	Referrer    string                 `json:"referrer,omitempty"`
	UTMSource   string                 `json:"utm_source,omitempty"`
	UTMMedium   string                 `json:"utm_medium,omitempty"`
	UTMCampaign string                 `json:"utm_campaign,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// FilterOptions represents analytics query filters
type FilterOptions struct {
	TenantID    string            `json:"tenant_id,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	EventTypes  []EventType       `json:"event_types,omitempty"`
	EventNames  []string          `json:"event_names,omitempty"`
	DateRange   *DateRange        `json:"date_range,omitempty"`
	Countries   []string          `json:"countries,omitempty"`
	Devices     []string          `json:"devices,omitempty"`
	Browsers    []string          `json:"browsers,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Search      string            `json:"search,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AggregationType represents different aggregation types
type AggregationType string

const (
	AggCount    AggregationType = "count"
	AggSum      AggregationType = "sum"
	AggAvg      AggregationType = "avg"
	AggMin      AggregationType = "min"
	AggMax      AggregationType = "max"
	AggUnique   AggregationType = "unique"
)

// Metric represents a metric definition
type Metric struct {
	Name         string          `json:"name"`
	Type         AggregationType `json:"type"`
	EventType    EventType       `json:"event_type"`
	FieldName    string          `json:"field_name,omitempty"`
	Filter       *FilterOptions  `json:"filter,omitempty"`
	Formula      string          `json:"formula,omitempty"`
	Format       string          `json:"format,omitempty"`
	Description  string          `json:"description,omitempty"`
}

// Dashboard represents an analytics dashboard
type Dashboard struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	TenantID    string         `json:"tenant_id,omitempty"`
	ProjectID   string         `json:"project_id,omitempty"`
	Widgets     []Widget       `json:"widgets"`
	Layout      DashboardLayout `json:"layout"`
	CreatedBy   string         `json:"created_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	IsPublic    bool           `json:"is_public"`
	RefreshRate int            `json:"refresh_rate"` // seconds
}

// Widget represents a dashboard widget
type Widget struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`       // chart, table, metric, text
	Title      string                 `json:"title"`
	Metrics    []Metric               `json:"metrics"`
	Filter     *FilterOptions         `json:"filter,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Position   WidgetPosition         `json:"position"`
	Size       WidgetSize             `json:"size"`
	DataSource string                 `json:"data_source"`
}

// WidgetPosition represents widget position
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WidgetSize represents widget size
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardLayout represents dashboard layout configuration
type DashboardLayout struct {
	Columns int `json:"columns"`
	Gap     int `json:"gap"`
	Padding int `json:"padding"`
}

// Report represents an analytics report
type Report struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Type        string                 `json:"type"` // real_time, daily, weekly, monthly
	Schedule    string                 `json:"schedule,omitempty"`
	Recipients  []string               `json:"recipients"`
	Queries     []Query                `json:"queries"`
	Format      string                 `json:"format"` // html, pdf, csv, json
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	IsActive    bool                   `json:"is_active"`
}

// Query represents an analytics query
type Query struct {
	Name     string         `json:"name"`
	Metrics  []Metric       `json:"metrics"`
	Filters  *FilterOptions `json:"filters,omitempty"`
	GroupBy  []string       `json:"group_by,omitempty"`
	OrderBy  string         `json:"order_by,omitempty"`
	Limit    int            `json:"limit,omitempty"`
}

// SearchQuery represents a search query
type SearchQuery struct {
	Query      string         `json:"query"`
	Filters    *FilterOptions `json:"filters,omitempty"`
	EntityType string         `json:"entity_type,omitempty"` // events, users, sessions
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	SortBy     string         `json:"sort_by,omitempty"`
	SortDir    string         `json:"sort_dir,omitempty"`
}

// SearchResult represents search results
type SearchResult struct {
	Total    int                    `json:"total"`
	Results  []map[string]interface{} `json:"results"`
	Facets   map[string][]string     `json:"facets,omitempty"`
	Suggestions []string            `json:"suggestions,omitempty"`
	Took     time.Duration          `json:"took"`
}

// Config represents analytics engine configuration
type Config struct {
	Storage          *storage.Config  `json:"storage"`
	RetentionPeriod  time.Duration    `json:"retention_period"`
	BatchSize        int              `json:"batch_size"`
	FlushInterval    time.Duration    `json:"flush_interval"`
	EnableRealTime   bool             `json:"enable_real_time"`
	SearchIndex      string           `json:"search_index"`
	MaxResults       int              `json:"max_results"`
	EnableSampling   bool             `json:"enable_sampling"`
	SamplingRate     float64          `json:"sampling_rate"`
}

// Engine represents the analytics engine
type Engine struct {
	config    *Config
	storage   storage.Engine
	ctx       context.Context
	cancel    context.CancelFunc
	eventChan chan *Event
}

// NewConfig creates a new analytics configuration
func NewConfig() *Config {
	return &Config{
		RetentionPeriod: 90 * 24 * time.Hour, // 90 days
		BatchSize:       1000,
		FlushInterval:   5 * time.Second,
		EnableRealTime:  true,
		SearchIndex:     "analytics_search",
		MaxResults:      10000,
		EnableSampling:  false,
		SamplingRate:    1.0,
	}
}

// NewEngine creates a new analytics engine
func NewEngine(config *Config, storageEngine storage.Engine) (*Engine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:    config,
		storage:   storageEngine,
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan *Event, 10000),
	}

	// Start event processor
	go engine.processEvents()

	// Start cleanup goroutine
	go engine.cleanupOldData()

	return engine, nil
}

// TrackEvent tracks an analytics event
func (e *Engine) TrackEvent(ctx context.Context, event *Event) error {
	// Generate ID if not provided
	if event.ID == "" {
		event.ID = e.generateEventID()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Enrich event data
	if err := e.enrichEvent(event); err != nil {
		return fmt.Errorf("failed to enrich event: %w", err)
	}

	// Apply sampling if enabled
	if e.config.EnableSampling && !e.shouldSample(event) {
		return nil
	}

	// Send to event processor
	select {
	case e.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Channel is full, store directly
		return e.storeEvent(ctx, event)
	}
}

// TrackPageView tracks a page view event
func (e *Engine) TrackPageView(ctx context.Context, tenantID, userID, sessionID, path, title string, properties map[string]interface{}) error {
	event := &Event{
		Type:       EventTypePageView,
		TenantID:   tenantID,
		UserID:     userID,
		SessionID:  sessionID,
		EventType:  "page_view",
		EventName:  "page_view",
		Properties: properties,
	}

	if properties == nil {
		event.Properties = make(map[string]interface{})
	}
	event.Properties["path"] = path
	event.Properties["title"] = title

	return e.TrackEvent(ctx, event)
}

// TrackAPIRequest tracks an API request
func (e *Engine) TrackAPIRequest(ctx context.Context, tenantID, userID, method, path string, statusCode int, duration time.Duration, properties map[string]interface{}) error {
	event := &Event{
		Type:       EventTypeAPIRequest,
		TenantID:   tenantID,
		UserID:     userID,
		EventType:  "api_request",
		EventName:  "api_request",
		Duration:   (*int64)(&duration.Milliseconds()),
		Properties: properties,
	}

	if properties == nil {
		event.Properties = make(map[string]interface{})
	}
	event.Properties["method"] = method
	event.Properties["path"] = path
	event.Properties["status_code"] = statusCode

	return e.TrackEvent(ctx, event)
}

// TrackError tracks an error event
func (e *Engine) TrackError(ctx context.Context, tenantID, userID, errorType, message string, stackTrace string, properties map[string]interface{}) error {
	event := &Event{
		Type:       EventTypeError,
		TenantID:   tenantID,
		UserID:     userID,
		EventType:  "error",
		EventName:  "error",
		Properties: properties,
	}

	if properties == nil {
		event.Properties = make(map[string]interface{})
	}
	event.Properties["error_type"] = errorType
	event.Properties["message"] = message
	event.Properties["stack_trace"] = stackTrace

	return e.TrackEvent(ctx, event)
}

// GetMetrics retrieves metrics data
func (e *Engine) GetMetrics(ctx context.Context, metrics []Metric, filters *FilterOptions) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, metric := range metrics {
		result, err := e.calculateMetric(ctx, metric, filters)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate metric %s: %w", metric.Name, err)
		}
		results[metric.Name] = result
	}

	return results, nil
}

// Search performs analytics search
func (e *Engine) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	start := time.Now()

	// Build search conditions
	conditions := make(map[string]interface{})

	if query.Filters != nil {
		if query.Filters.TenantID != "" {
			conditions["tenant_id"] = query.Filters.TenantID
		}
		if query.Filters.ProjectID != "" {
			conditions["project_id"] = query.Filters.ProjectID
		}
		if query.Filters.UserID != "" {
			conditions["user_id"] = query.Filters.UserID
		}
	}

	// Add search text condition
	if query.Query != "" {
		// Full-text search implementation
		conditions["search"] = query.Query
	}

	// Determine table to search
	table := "events"
	if query.EntityType != "" {
		switch query.EntityType {
		case "users":
			table = "users"
		case "sessions":
			table = "sessions"
		}
	}

	// Query storage
	queryOptions := &storage.QueryOptions{
		Limit:  query.Limit,
		Offset: query.Offset,
		Where:  conditions,
	}

	result, err := e.storage.Query(ctx, table, queryOptions)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to search results
	searchResults := make([]map[string]interface{}, len(result.Records))
	for i, record := range result.Records {
		searchResults[i] = record.Data
	}

	// Extract facets and suggestions
	facets := e.extractFacets(result.Records, query.Query)
	suggestions := e.generateSuggestions(query.Query)

	return &SearchResult{
		Total:       result.Total,
		Results:     searchResults,
		Facets:      facets,
		Suggestions: suggestions,
		Took:        time.Since(start),
	}, nil
}

// GetRealTimeStats retrieves real-time statistics
func (e *Engine) GetRealTimeStats(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	now := time.Now()
	minutes5Ago := now.Add(-5 * time.Minute)
	hour1Ago := now.Add(-1 * time.Hour)
	day1Ago := now.Add(-24 * time.Hour)

	stats := make(map[string]interface{})

	// Active users (last 5 minutes)
	activeUsers, _ := e.getActiveUsers(ctx, tenantID, minutes5Ago)
	stats["active_users_5m"] = activeUsers

	// Active users (last hour)
	activeUsers1h, _ := e.getActiveUsers(ctx, tenantID, hour1Ago)
	stats["active_users_1h"] = activeUsers1h

	// Page views (last hour)
	pageViews1h, _ := e.getEventCount(ctx, tenantID, EventTypePageView, hour1Ago, now)
	stats["page_views_1h"] = pageViews1h

	// API requests (last hour)
	apiRequests1h, _ := e.getEventCount(ctx, tenantID, EventTypeAPIRequest, hour1Ago, now)
	stats["api_requests_1h"] = apiRequests1h

	// Error rate (last hour)
	errors1h, _ := e.getEventCount(ctx, tenantID, EventTypeError, hour1Ago, now)
	totalEvents1h, _ := e.getEventCount(ctx, tenantID, "", hour1Ago, now)
	errorRate := float64(0)
	if totalEvents1h > 0 {
		errorRate = float64(errors1h) / float64(totalEvents1h) * 100
	}
	stats["error_rate_1h"] = errorRate

	// Top pages (last 24 hours)
	topPages, _ := e.getTopPages(ctx, tenantID, day1Ago, now, 10)
	stats["top_pages_24h"] = topPages

	// Response time average (last hour)
	avgResponseTime, _ := e.getAverageResponseTime(ctx, tenantID, hour1Ago, now)
	stats["avg_response_time_1h"] = avgResponseTime

	return stats, nil
}

// CreateDashboard creates a new dashboard
func (e *Engine) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	if dashboard.ID == "" {
		dashboard.ID = e.generateDashboardID()
	}
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	return e.saveDashboard(ctx, dashboard)
}

// GetDashboard retrieves a dashboard
func (e *Engine) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	return e.loadDashboard(ctx, id)
}

// UpdateDashboard updates an existing dashboard
func (e *Engine) UpdateDashboard(ctx context.Context, dashboard *Dashboard) error {
	dashboard.UpdatedAt = time.Now()
	return e.saveDashboard(ctx, dashboard)
}

// DeleteDashboard deletes a dashboard
func (e *Engine) DeleteDashboard(ctx context.Context, id string) error {
	return e.storage.Delete(ctx, "dashboards", id)
}

// GetDashboardData retrieves data for all widgets in a dashboard
func (e *Engine) GetDashboardData(ctx context.Context, dashboard *Dashboard, filters *FilterOptions) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	for _, widget := range dashboard.Widgets {
		widgetData, err := e.GetMetrics(ctx, widget.Metrics, filters)
		if err != nil {
			return nil, fmt.Errorf("failed to get data for widget %s: %w", widget.ID, err)
		}
		data[widget.ID] = widgetData
	}

	return data, nil
}

// Private methods

func (e *Engine) processEvents() {
	batch := make([]*Event, 0, e.config.BatchSize)
	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-e.eventChan:
			batch = append(batch, event)
			if len(batch) >= e.config.BatchSize {
				e.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				e.flushBatch(batch)
				batch = batch[:0]
			}

		case <-e.ctx.Done():
			// Flush remaining events before exiting
			if len(batch) > 0 {
				e.flushBatch(batch)
			}
			return
		}
	}
}

func (e *Engine) flushBatch(events []*Event) {
	ctx := context.Background()
	for _, event := range events {
		if err := e.storeEvent(ctx, event); err != nil {
			// Log error but continue processing other events
			fmt.Printf("Failed to store event %s: %v\n", event.ID, err)
		}
	}
}

func (e *Engine) storeEvent(ctx context.Context, event *Event) error {
	data := map[string]interface{}{
		"id":           event.ID,
		"type":         event.Type,
		"tenant_id":    event.TenantID,
		"project_id":   event.ProjectID,
		"user_id":      event.UserID,
		"session_id":   event.SessionID,
		"event_type":   event.EventType,
		"event_name":   event.EventName,
		"properties":   event.Properties,
		"timestamp":    event.Timestamp,
		"duration":     event.Duration,
		"user_agent":   event.UserAgent,
		"ip":           event.IP,
		"country":      event.Country,
		"city":         event.City,
		"device":       event.Device,
		"browser":      event.Browser,
		"os":           event.OS,
		"referrer":     event.Referrer,
		"utm_source":   event.UTMSource,
		"utm_medium":   event.UTMMedium,
		"utm_campaign": event.UTMCampaign,
		"tags":         event.Tags,
	}

	_, err := e.storage.Create(ctx, "events", data)
	return err
}

func (e *Engine) enrichEvent(event *Event) error {
	// Parse user agent if provided
	if event.UserAgent != "" {
		event.Device = parseDevice(event.UserAgent)
		event.Browser = parseBrowser(event.UserAgent)
		event.OS = parseOS(event.UserAgent)
	}

	// Extract country/city from IP (would require IP geolocation service)
	if event.IP != "" {
		event.Country = lookupCountry(event.IP)
		event.City = lookupCity(event.IP)
	}

	return nil
}

func (e *Engine) shouldSample(event *Event) bool {
	// Always sample errors and important events
	if event.Type == EventTypeError || event.Type == EventTypeConversion {
		return true
	}

	// Use random sampling for other events
	return e.randomFloat() < e.config.SamplingRate
}

func (e *Engine) calculateMetric(ctx context.Context, metric Metric, filters *FilterOptions) (interface{}, error) {
	// Build query conditions based on metric and filters
	conditions := make(map[string]interface{})

	if metric.Filter != nil {
		e.applyFilters(conditions, metric.Filter)
	}
	if filters != nil {
		e.applyFilters(conditions, filters)
	}

	conditions["event_type"] = metric.EventType

	queryOptions := &storage.QueryOptions{
		Where: conditions,
	}

	switch metric.Type {
	case AggCount:
		result, err := e.storage.Query(ctx, "events", queryOptions)
		if err != nil {
			return 0, err
		}
		return result.Total, nil

	case AggSum, AggAvg, AggMin, AggMax:
		if metric.FieldName == "" {
			return nil, fmt.Errorf("field_name required for aggregation type %s", metric.Type)
		}
		// Implement aggregation logic
		return e.calculateNumericMetric(ctx, metric, queryOptions)

	case AggUnique:
		return e.calculateUniqueCount(ctx, metric, queryOptions)

	default:
		return nil, fmt.Errorf("unsupported aggregation type: %s", metric.Type)
	}
}

func (e *Engine) applyFilters(conditions map[string]interface{}, filters *FilterOptions) {
	if filters.TenantID != "" {
		conditions["tenant_id"] = filters.TenantID
	}
	if filters.ProjectID != "" {
		conditions["project_id"] = filters.ProjectID
	}
	if filters.UserID != "" {
		conditions["user_id"] = filters.UserID
	}
	if len(filters.EventTypes) > 0 {
		conditions["event_types"] = filters.EventTypes
	}
	if len(filters.EventNames) > 0 {
		conditions["event_names"] = filters.EventNames
	}
	if filters.DateRange != nil {
		conditions["date_range"] = map[string]interface{}{
			"start": filters.DateRange.Start,
			"end":   filters.DateRange.End,
		}
	}
}

// Utility functions

func (e *Engine) generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

func (e *Engine) generateDashboardID() string {
	return fmt.Sprintf("dashboard_%d", time.Now().UnixNano())
}

func (e *Engine) extractFacets(records []storage.Record, query string) map[string][]string {
	// Implement facet extraction logic
	return make(map[string][]string)
}

func (e *Engine) generateSuggestions(query string) []string {
	// Implement suggestion logic
	return []string{}
}

func (e *Engine) getActiveUsers(ctx context.Context, tenantID string, since time.Time) (int, error) {
	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"date_range": map[string]interface{}{
			"start": since,
		},
	}

	queryOptions := &storage.QueryOptions{
		Where: conditions,
	}

	result, err := e.storage.Query(ctx, "events", queryOptions)
	if err != nil {
		return 0, err
	}

	// Count unique users
	users := make(map[string]bool)
	for _, record := range result.Records {
		if userID, ok := record.Data["user_id"].(string); ok && userID != "" {
			users[userID] = true
		}
	}

	return len(users), nil
}

func (e *Engine) getEventCount(ctx context.Context, tenantID string, eventType EventType, start, end time.Time) (int, error) {
	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"date_range": map[string]interface{}{
			"start": start,
			"end":   end,
		},
	}

	if eventType != "" {
		conditions["type"] = eventType
	}

	queryOptions := &storage.QueryOptions{
		Where: conditions,
	}

	result, err := e.storage.Query(ctx, "events", queryOptions)
	if err != nil {
		return 0, err
	}

	return result.Total, nil
}

func (e *Engine) getTopPages(ctx context.Context, tenantID string, start, end time.Time, limit int) []map[string]interface{} {
	// Implement top pages logic
	return []map[string]interface{}{}
}

func (e *Engine) getAverageResponseTime(ctx context.Context, tenantID string, start, end time.Time) float64 {
	// Implement average response time logic
	return 0.0
}

func (e *Engine) saveDashboard(ctx context.Context, dashboard *Dashboard) error {
	data := map[string]interface{}{
		"id":          dashboard.ID,
		"name":        dashboard.Name,
		"description": dashboard.Description,
		"tenant_id":   dashboard.TenantID,
		"project_id":  dashboard.ProjectID,
		"widgets":     dashboard.Widgets,
		"layout":      dashboard.Layout,
		"created_by":  dashboard.CreatedBy,
		"created_at":  dashboard.CreatedAt,
		"updated_at":  dashboard.UpdatedAt,
		"is_public":   dashboard.IsPublic,
		"refresh_rate": dashboard.RefreshRate,
	}

	_, err := e.storage.Create(ctx, "dashboards", data)
	return err
}

func (e *Engine) loadDashboard(ctx context.Context, id string) (*Dashboard, error) {
	record, err := e.storage.Get(ctx, "dashboards", id)
	if err != nil {
		return nil, err
	}

	// Convert record to Dashboard
	dashboard := &Dashboard{}
	// Implement proper mapping
	return dashboard, nil
}

func (e *Engine) cleanupOldData() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.performCleanup()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *Engine) performCleanup() {
	ctx := context.Background()
	cutoff := time.Now().Add(-e.config.RetentionPeriod)

	conditions := map[string]interface{}{
		"date_range": map[string]interface{}{
			"end": cutoff,
		},
	}

	queryOptions := &storage.QueryOptions{
		Where:  conditions,
		Limit:  10000, // Process in batches
	}

	result, err := e.storage.Query(ctx, "events", queryOptions)
	if err != nil {
		return
	}

	for _, record := range result.Records {
		e.storage.Delete(ctx, "events", record.ID)
	}
}

// Helper parsing functions
func parseDevice(userAgent string) string {
	// Implement device parsing
	if strings.Contains(strings.ToLower(userAgent), "mobile") {
		return "mobile"
	}
	if strings.Contains(strings.ToLower(userAgent), "tablet") {
		return "tablet"
	}
	return "desktop"
}

func parseBrowser(userAgent string) string {
	// Implement browser parsing
	if strings.Contains(userAgent, "Chrome") {
		return "Chrome"
	}
	if strings.Contains(userAgent, "Firefox") {
		return "Firefox"
	}
	if strings.Contains(userAgent, "Safari") {
		return "Safari"
	}
	return "Other"
}

func parseOS(userAgent string) string {
	// Implement OS parsing
	if strings.Contains(userAgent, "Windows") {
		return "Windows"
	}
	if strings.Contains(userAgent, "Mac") {
		return "macOS"
	}
	if strings.Contains(userAgent, "Linux") {
		return "Linux"
	}
	return "Other"
}

func lookupCountry(ip string) string {
	// Implement IP to country lookup
	return "Unknown"
}

func lookupCity(ip string) string {
	// Implement IP to city lookup
	return "Unknown"
}

func (e *Engine) randomFloat() float64 {
	// Implement random float generation
	return 1.0 // Placeholder
}

func (e *Engine) calculateNumericMetric(ctx context.Context, metric Metric, queryOptions *storage.QueryOptions) (interface{}, error) {
	// Implement numeric metric calculation
	return 0, nil
}

func (e *Engine) calculateUniqueCount(ctx context.Context, metric Metric, queryOptions *storage.QueryOptions) (interface{}, error) {
	// Implement unique count calculation
	return 0, nil
}

func (e *Engine) Close() error {
	if e.cancel != nil {
		e.cancel()
	}
	close(e.eventChan)
	return nil
}