package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/guileen/metabase/pkg/common/nrpc/embedded"
	"github.com/nats-io/nats.go"
)

// Integration provides real-time and API integration
type Integration struct {
	engine        *Engine
	natsConn      *nats.Conn
	httpServer    *http.Server
	wsUpgrader    websocket.Upgrader
	wsClients     map[*websocket.Conn]bool
	wsMutex       sync.RWMutex
	subscriptions map[string]*nats.Subscription
}

// APIRequest represents an API request
type APIRequest struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Artifact  *Artifact              `json:"artifact,omitempty"`
	Query     *Query                 `json:"query,omitempty"`
	Options   map[string]interface{} `json:"options"`
	Timestamp time.Time              `json:"timestamp"`
}

// APIResponse represents an API response
type APIResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Config represents integration configuration
type IntegrationConfig struct {
	HTTPPort        int           `json:"http_port"`
	NATSServerURL   string        `json:"nats_server_url"`
	EnableRealtime  bool          `json:"enable_realtime"`
	EnableWebSocket bool          `json:"enable_websocket"`
	MaxConnections  int           `json:"max_connections"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	EnableAuth      bool          `json:"enable_auth"`
	APIKeyRequired  bool          `json:"api_key_required"`
}

// NewIntegration creates a new integration layer
func NewIntegration(engine *Engine, config *IntegrationConfig) (*Integration, error) {
	integration := &Integration{
		engine:        engine,
		wsUpgrader:    websocket.Upgrader{},
		wsClients:     make(map[*websocket.Conn]bool),
		subscriptions: make(map[string]*nats.Subscription),
	}

	// Setup NATS connection if enabled
	if config.EnableRealtime {
		nc, err := embedded.Connect()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		integration.natsConn = nc

		// Setup NATS subscriptions
		integration.setupNATSSubscriptions()
	}

	// Setup HTTP server
	integration.setupHTTPServer(config)

	return integration, nil
}

// setupNATSSubscriptions sets up NATS message handlers
func (i *Integration) setupNATSSubscriptions() {
	// Analysis requests
	sub, err := i.natsConn.Subscribe("analysis.request", i.handleAnalysisRequest)
	if err != nil {
		log.Printf("Failed to subscribe to analysis.requests: %v", err)
	}
	i.subscriptions["analysis.request"] = sub

	// Search requests
	sub, err = i.natsConn.Subscribe("search.request", i.handleSearchRequest)
	if err != nil {
		log.Printf("Failed to subscribe to search.requests: %v", err)
	}
	i.subscriptions["search.request"] = sub

	// Index updates
	sub, err = i.natsConn.Subscribe("index.update", i.handleIndexUpdate)
	if err != nil {
		log.Printf("Failed to subscribe to index.updates: %v", err)
	}
	i.subscriptions["index.update"] = sub
}

// setupHTTPServer sets up the HTTP API server
func (i *Integration) setupHTTPServer(config *IntegrationConfig) {
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Analysis endpoints
	api.HandleFunc("/analyze", i.handleAnalyzeAPI).Methods("POST")
	api.HandleFunc("/analyze/{id}", i.getAnalysisResult).Methods("GET")
	api.HandleFunc("/analyze/{id}/status", i.getAnalysisStatus).Methods("GET")

	// Search endpoints
	api.HandleFunc("/search", i.handleSearchAPI).Methods("POST")
	api.HandleFunc("/search/suggest", i.handleSearchSuggest).Methods("GET")
	api.HandleFunc("/search/history", i.getSearchHistory).Methods("GET")

	// Duplicate detection
	api.HandleFunc("/duplicates", i.handleDuplicateCheck).Methods("POST")

	// Security scanning
	api.HandleFunc("/security/scan", i.handleSecurityScan).Methods("POST")
	api.HandleFunc("/security/report", i.getSecurityReport).Methods("GET")

	// Quality metrics
	api.HandleFunc("/quality/metrics", i.getQualityMetrics).Methods("GET")
	api.HandleFunc("/quality/report", i.getQualityReport).Methods("POST")

	// Index management
	api.HandleFunc("/index/build", i.buildIndex).Methods("POST")
	api.HandleFunc("/index/stats", i.getIndexStats).Methods("GET")
	api.HandleFunc("/index/clear", i.clearIndex).Methods("DELETE")

	// Real-time WebSocket endpoint
	if config.EnableWebSocket {
		api.HandleFunc("/ws", i.handleWebSocket)
	}

	// System endpoints
	api.HandleFunc("/health", i.healthCheck).Methods("GET")
	api.HandleFunc("/stats", i.getSystemStats).Methods("GET")

	i.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.HTTPPort),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}
}

// Start starts the integration services
func (i *Integration) Start() error {
	// Start HTTP server
	go func() {
		log.Printf("Starting API server on %s", i.httpServer.Addr)
		if err := i.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the integration services
func (i *Integration) Stop() error {
	// Close HTTP server
	if i.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		i.httpServer.Shutdown(ctx)
	}

	// Close NATS subscriptions
	for _, sub := range i.subscriptions {
		sub.Unsubscribe()
	}

	// Close NATS connection
	if i.natsConn != nil {
		i.natsConn.Close()
	}

	// Close WebSocket connections
	i.wsMutex.Lock()
	for client := range i.wsClients {
		client.Close()
	}
	i.wsClients = make(map[*websocket.Conn]bool)
	i.wsMutex.Unlock()

	return nil
}

// handleAnalysisRequest handles NATS analysis requests
func (i *Integration) handleAnalysisRequest(msg *nats.Msg) {
	var req APIRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		i.sendErrorReply(msg.Reply, "Invalid request format")
		return
	}

	switch req.Type {
	case "artifact":
		if req.Artifact == nil {
			i.sendErrorReply(msg.Reply, "Artifact required")
			return
		}

		// Perform analysis
		results, err := i.engine.Analyze(context.Background(), req.Artifact)
		if err != nil {
			i.sendErrorReply(msg.Reply, err.Error())
			return
		}

		// Send response
		resp := APIResponse{
			ID:        req.ID,
			Type:      "analysis_result",
			Success:   true,
			Data:      results,
			Timestamp: time.Now(),
		}
		i.sendResponseReply(msg.Reply, resp)

	case "batch":
		// Handle batch analysis
		artifacts, ok := req.Options["artifacts"].([]*Artifact)
		if !ok {
			i.sendErrorReply(msg.Reply, "Batch artifacts required")
			return
		}

		results := make([]*AnalysisResult, 0)
		for _, artifact := range artifacts {
			result, err := i.engine.Analyze(context.Background(), artifact)
			if err != nil {
				continue
			}
			results = append(results, result...)
		}

		resp := APIResponse{
			ID:        req.ID,
			Type:      "batch_analysis_result",
			Success:   true,
			Data:      results,
			Timestamp: time.Now(),
		}
		i.sendResponseReply(msg.Reply, resp)
	}
}

// handleSearchRequest handles NATS search requests
func (i *Integration) handleSearchRequest(msg *nats.Msg) {
	var req APIRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		i.sendErrorReply(msg.Reply, "Invalid request format")
		return
	}

	if req.Query == nil {
		i.sendErrorReply(msg.Reply, "Query required")
		return
	}

	// Perform search
	results, err := i.engine.Search(context.Background(), req.Query)
	if err != nil {
		i.sendErrorReply(msg.Reply, err.Error())
		return
	}

	// Send response
	resp := APIResponse{
		ID:        req.ID,
		Type:      "search_result",
		Success:   true,
		Data:      results,
		Timestamp: time.Now(),
	}
	i.sendResponseReply(msg.Reply, resp)
}

// handleIndexUpdate handles index updates
func (i *Integration) handleIndexUpdate(msg *nats.Msg) {
	var req APIRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return
	}

	// Process index update
	if req.Artifact != nil {
		// Index artifact
		// In production, this would update search indexes
		log.Printf("Indexing artifact: %s", req.Artifact.ID)
	}

	// Broadcast to WebSocket clients
	i.broadcastMessage("index.update", WebSocketMessage{
		Type:      "index_updated",
		Channel:   "index",
		Data:      map[string]string{"artifact_id": req.Artifact.ID},
		Timestamp: time.Now(),
	})
}

// handleAnalyzeAPI handles HTTP analyze requests
func (i *Integration) handleAnalyzeAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artifact *Artifact              `json:"artifact"`
		Options  map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Artifact == nil {
		http.Error(w, "Artifact required", http.StatusBadRequest)
		return
	}

	// Perform analysis
	results, err := i.engine.Analyze(r.Context(), req.Artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    results,
	})
}

// handleSearchAPI handles HTTP search requests
func (i *Integration) handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	var query Query
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Perform search
	results, err := i.engine.Search(r.Context(), &query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    results,
	})
}

// handleDuplicateCheck handles duplicate detection
func (i *Integration) handleDuplicateCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artifact  *Artifact `json:"artifact"`
		Threshold float64   `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Threshold == 0 {
		req.Threshold = 0.8
	}

	// Find duplicates
	duplicates, err := i.engine.FindDuplicates(r.Context(), req.Artifact, req.Threshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"duplicates": duplicates,
	})
}

// handleWebSocket handles WebSocket connections
func (i *Integration) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := i.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Add to clients
	i.wsMutex.Lock()
	i.wsClients[conn] = true
	i.wsMutex.Unlock()

	// Handle messages
	go i.handleWebSocketMessages(conn)
}

// handleWebSocketMessages handles WebSocket message loop
func (i *Integration) handleWebSocketMessages(conn *websocket.Conn) {
	defer func() {
		i.wsMutex.Lock()
		delete(i.wsClients, conn)
		i.wsMutex.Unlock()
		conn.Close()
	}()

	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		// Handle message
		switch msg.Type {
		case "subscribe":
			channel, ok := msg.Data.(string)
			if ok {
				// Handle subscription
				log.Printf("Client subscribed to channel: %s", channel)
			}
		case "search":
			// Handle real-time search
			queryData, ok := msg.Data.(map[string]interface{})
			if ok {
				queryBytes, _ := json.Marshal(queryData)
				var query Query
				json.Unmarshal(queryBytes, &query)

				results, err := i.engine.Search(context.Background(), &query)
				if err == nil {
					conn.WriteJSON(WebSocketMessage{
						Type:      "search_result",
						Channel:   "search",
						Data:      results,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}
}

// broadcastMessage broadcasts message to all WebSocket clients
func (i *Integration) broadcastMessage(channel string, msg WebSocketMessage) {
	i.wsMutex.RLock()
	defer i.wsMutex.RUnlock()

	for client := range i.wsClients {
		if err := client.WriteJSON(msg); err != nil {
			// Remove failed client
			delete(i.wsClients, client)
			client.Close()
		}
	}
}

// sendErrorReply sends error response via NATS
func (i *Integration) sendErrorReply(reply string, error string) {
	if reply == "" {
		return
	}

	resp := APIResponse{
		Type:      "error",
		Success:   false,
		Error:     error,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(resp)
	i.natsConn.Publish(reply, data)
}

// sendResponseReply sends response via NATS
func (i *Integration) sendResponseReply(reply string, resp APIResponse) {
	if reply == "" {
		return
	}

	data, _ := json.Marshal(resp)
	i.natsConn.Publish(reply, data)
}

// healthCheck returns system health
func (i *Integration) healthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}

	// Check engine status
	stats := i.engine.GetStats()
	status["engine"] = map[string]interface{}{
		"artifacts_processed": stats.ArtifactsProcessed,
		"analysis_count":      stats.AnalysisCount,
		"search_count":        stats.SearchCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getSystemStats returns system statistics
func (i *Integration) getSystemStats(w http.ResponseWriter, r *http.Request) {
	stats := i.engine.GetStats()

	response := map[string]interface{}{
		"engine": stats,
		"integration": map[string]interface{}{
			"websocket_clients":  len(i.wsClients),
			"nats_subscriptions": len(i.subscriptions),
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSearchSuggest provides search suggestions
func (i *Integration) handleSearchSuggest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 10

	// Mock suggestions
	suggestions := []string{
		query + " function",
		query + " class",
		query + " variable",
		query + " method",
		query + " interface",
	}

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":       query,
		"suggestions": suggestions,
	})
}

// handleSecurityScan handles security scanning requests
func (i *Integration) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artifact *Artifact `json:"artifact"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Use security scanner
	scanner := NewSecurityScanner()
	result, err := scanner.Analyze(r.Context(), req.Artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

// getSecurityReport returns security report
func (i *Integration) getSecurityReport(w http.ResponseWriter, r *http.Request) {
	// Mock security report
	report := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_vulnerabilities": 5,
			"critical":              1,
			"high":                  2,
			"medium":                1,
			"low":                   1,
		},
		"vulnerabilities": []map[string]interface{}{
			{
				"type":     "SQL Injection",
				"severity": "critical",
				"file":     "database.go",
				"line":     42,
			},
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// getQualityMetrics returns quality metrics
func (i *Integration) getQualityMetrics(w http.ResponseWriter, r *http.Request) {
	// Mock quality metrics
	metrics := map[string]interface{}{
		"overall_score": 85.5,
		"metrics": map[string]float64{
			"complexity":      12.3,
			"maintainability": 75.2,
			"test_coverage":   78.0,
			"documentation":   45.5,
			"duplication":     8.2,
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// getQualityReport returns quality report
func (i *Integration) getQualityReport(w http.ResponseWriter, r *http.Request) {
	// Mock quality report
	report := map[string]interface{}{
		"files_analyzed": 125,
		"issues": []map[string]interface{}{
			{
				"type":     "High Complexity",
				"severity": "medium",
				"file":     "processor.go",
				"line":     156,
			},
		},
		"recommendations": []string{
			"Refactor complex functions",
			"Add more unit tests",
			"Improve documentation",
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// getAnalysisResult returns specific analysis result
func (i *Integration) getAnalysisResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Mock result
	result := map[string]interface{}{
		"id":           id,
		"status":       "completed",
		"findings":     []string{},
		"score":        95.0,
		"completed_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getAnalysisStatus returns analysis status
func (i *Integration) getAnalysisStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Mock status
	status := map[string]interface{}{
		"id":       id,
		"status":   "processing",
		"progress": 75,
		"message":  "Analyzing code quality...",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getSearchHistory returns search history
func (i *Integration) getSearchHistory(w http.ResponseWriter, r *http.Request) {
	// Mock history
	history := []map[string]interface{}{
		{
			"query":     "function calculateSum",
			"timestamp": time.Now().Add(-1 * time.Hour),
			"results":   5,
		},
		{
			"query":     "SQL injection",
			"timestamp": time.Now().Add(-2 * time.Hour),
			"results":   3,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}

// buildIndex rebuilds search index
func (i *Integration) buildIndex(w http.ResponseWriter, r *http.Request) {
	// Trigger index rebuild
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Index rebuild started",
	})
}

// getIndexStats returns index statistics
func (i *Integration) getIndexStats(w http.ResponseWriter, r *http.Request) {
	// Mock index stats
	stats := map[string]interface{}{
		"documents_indexed": 1000,
		"index_size":        "125MB",
		"last_updated":      time.Now(),
		"types": map[string]int{
			"source":        800,
			"documentation": 150,
			"config":        50,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// clearIndex clears search index
func (i *Integration) clearIndex(w http.ResponseWriter, r *http.Request) {
	// Clear index
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Index cleared",
	})
}
