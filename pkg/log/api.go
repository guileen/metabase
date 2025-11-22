package log

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// API provides HTTP endpoints for log management
type API struct {
	storage *LogStorage
}

// NewAPI creates a new log API instance
func NewAPI(storage *LogStorage) *API {
	return &API{
		storage: storage,
	}
}

// RegisterRoutes registers the log API routes
func (api *API) RegisterRoutes(r chi.Router) {
	r.Route("/logs", func(r chi.Router) {
		r.Get("/", api.QueryLogs)
		r.Get("/stats", api.GetStats)
		r.Get("/services", api.GetServices)
		r.Get("/components", api.GetComponents)
		r.Get("/levels", api.GetLevels)
	})
}

// QueryLogs handles log query requests
func (api *API) QueryLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := &LogQuery{}

	// Time range
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	// Filters
	if levels := r.URL.Query()["levels"]; len(levels) > 0 {
		query.Levels = levels
	}

	if services := r.URL.Query()["services"]; len(services) > 0 {
		query.Services = services
	}

	if components := r.URL.Query()["components"]; len(components) > 0 {
		query.Components = components
	}

	if userIDs := r.URL.Query()["user_ids"]; len(userIDs) > 0 {
		query.UserIDs = userIDs
	}

	if requestIDs := r.URL.Query()["request_ids"]; len(requestIDs) > 0 {
		query.RequestIDs = requestIDs
	}

	// Search
	query.Search = r.URL.Query().Get("search")
	query.Method = r.URL.Query().Get("method")
	query.Path = r.URL.Query().Get("path")

	// Status range
	if minStatusStr := r.URL.Query().Get("min_status"); minStatusStr != "" {
		if minStatus, err := strconv.Atoi(minStatusStr); err == nil {
			query.MinStatus = minStatus
		}
	}

	if maxStatusStr := r.URL.Query().Get("max_status"); maxStatusStr != "" {
		if maxStatus, err := strconv.Atoi(maxStatusStr); err == nil {
			query.MaxStatus = maxStatus
		}
	}

	// Pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	}

	// Sorting
	query.OrderBy = r.URL.Query().Get("order_by")
	query.OrderDir = r.URL.Query().Get("order_dir")

	// Query logs
	logs, err := api.storage.QueryLogs(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query logs: %v", err), http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	countQuery := *query
	countQuery.Limit = 0
	countQuery.Offset = 0
	totalLogs, err := api.storage.QueryLogs(ctx, &countQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get total count: %v", err), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"logs":  logs,
		"total": len(totalLogs),
		"query": query,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetStats handles log statistics requests
func (api *API) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse time range
	var timeRange *time.Time
	if timeStr := r.URL.Query().Get("time_range"); timeStr != "" {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			timeRange = &t
		}
	}

	// Get statistics
	stats, err := api.storage.GetLogStats(ctx, timeRange)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get log statistics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetServices returns the list of services that have logs
func (api *API) GetServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := &LogQuery{
		Limit: 1000, // Get all unique services
	}

	logs, err := api.storage.QueryLogs(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get services: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract unique services
	services := make(map[string]bool)
	for _, log := range logs {
		if log.Service != "" {
			services[log.Service] = true
		}
	}

	// Convert to slice
	serviceList := make([]string, 0, len(services))
	for service := range services {
		serviceList = append(serviceList, service)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"services": serviceList,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetComponents returns the list of components that have logs
func (api *API) GetComponents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := &LogQuery{
		Limit: 1000, // Get all unique components
	}

	logs, err := api.storage.QueryLogs(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get components: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract unique components
	components := make(map[string]bool)
	for _, log := range logs {
		if log.Component != "" {
			components[log.Component] = true
		}
	}

	// Convert to slice
	componentList := make([]string, 0, len(components))
	for component := range components {
		componentList = append(componentList, component)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"components": componentList,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetLevels returns the list of log levels
func (api *API) GetLevels(w http.ResponseWriter, r *http.Request) {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"levels": levels,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// LogQueryResponse represents the response structure for log queries
type LogQueryResponse struct {
	Logs  []*StoredLogEntry `json:"logs"`
	Total int               `json:"total"`
	Query *LogQuery         `json:"query"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// Helper function to write error responses
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	})
}

// Helper function to write success responses
func writeSuccessResponse(w http.ResponseWriter, data interface{}, message ...string) {
	response := SuccessResponse{
		Data: data,
	}
	if len(message) > 0 {
		response.Message = message[0]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateQuery validates a log query
func validateQuery(query *LogQuery) error {
	// Validate time range
	if query.StartTime != nil && query.EndTime != nil {
		if query.StartTime.After(*query.EndTime) {
			return fmt.Errorf("start_time must be before end_time")
		}
	}

	// Validate limit
	if query.Limit < 0 || query.Limit > 1000 {
		return fmt.Errorf("limit must be between 0 and 1000")
	}

	// Validate offset
	if query.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}

	// Validate status range
	if query.MinStatus < 0 || query.MinStatus > 599 {
		return fmt.Errorf("min_status must be between 0 and 599")
	}

	if query.MaxStatus < 0 || query.MaxStatus > 599 {
		return fmt.Errorf("max_status must be between 0 and 599")
	}

	if query.MinStatus > 0 && query.MaxStatus > 0 && query.MinStatus > query.MaxStatus {
		return fmt.Errorf("min_status must be less than or equal to max_status")
	}

	// Validate order direction
	if query.OrderDir != "" && query.OrderDir != "asc" && query.OrderDir != "desc" {
		return fmt.Errorf("order_dir must be 'asc' or 'desc'")
	}

	// Validate order by
	if query.OrderBy != "" {
		validOrderFields := []string{"timestamp", "level", "duration_ms", "status", "path", "service", "component"}
		valid := false
		for _, field := range validOrderFields {
			if query.OrderBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("order_by must be one of: timestamp, level, duration_ms, status, path, service, component")
		}
	}

	return nil
}
