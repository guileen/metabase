package console

import ("context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/infra/storage")

// LogLevel represents the log level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry represents a log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	Method    string                 `json:"method"`
	Duration  time.Duration          `json:"duration"`
	Status    int                    `json:"status"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"user_id"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
}

// MetricValue represents a metric value
type MetricValue struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// Stats represents system statistics
type Stats struct {
	QPS         float64                 `json:"qps"`
	Latency     LatencyStats            `json:"latency"`
	ErrorRate   float64                 `json:"error_rate"`
	Throughput  map[string]float64      `json:"throughput"`
	System      SystemStats             `json:"system"`
	Storage     map[string]interface{}  `json:"storage"`
	Custom      map[string]interface{}  `json:"custom"`
}

// LatencyStats represents latency statistics
type LatencyStats struct {
	P50 time.Duration `json:"p50"`
	P90 time.Duration `json:"p90"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
	Avg time.Duration `json:"avg"`
}

// SystemStats represents system resource statistics
type SystemStats struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Disk   float64 `json:"disk"`
	NetIO  float64 `json:"net_io"`
}

// Config represents console configuration
type Config struct {
	Port       string `json:"port"`
	Host       string `json:"host"`
	Storage    *storage.Engine `json:"-"`
	DevMode    bool   `json:"dev_mode"`
	LogLevel   LogLevel `json:"log_level"`
	MaxLogs    int    `json:"max_logs"`
	MetricsTTL time.Duration `json:"metrics_ttl"`
}

// Server represents the console server
type Server struct {
	config     *Config
	logs       []LogEntry
	metrics    map[string][]MetricValue
	stats      *Stats
	mu         sync.RWMutex
	httpServer *http.Server
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewConfig creates a new console configuration with defaults
func NewConfig() *Config {
	return &Config{
		Port:       "7610",
		Host:       "localhost",
		DevMode:    true,
		LogLevel:   LogLevelInfo,
		MaxLogs:    10000,
		MetricsTTL: time.Hour,
	}
}

// NewServer creates a new console server
func NewServer(config *Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:  config,
		logs:    make([]LogEntry, 0, config.MaxLogs),
		metrics: make(map[string][]MetricValue),
		stats: &Stats{
			Throughput: make(map[string]float64),
			Custom:     make(map[string]interface{}),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize statistics
	server.initStats()

	return server
}

// Start starts the console server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register API routes
	mux.HandleFunc("/api/console/logs", s.handleLogs)
	mux.HandleFunc("/api/console/metrics", s.handleMetrics)
	mux.HandleFunc("/api/console/stats", s.handleStats)
	mux.HandleFunc("/api/console/search", s.handleSearch)
	mux.HandleFunc("/api/console/config", s.handleConfig)

	// Register admin routes
	mux.HandleFunc("/api/console/logs/", s.handleLogDetails)
	mux.HandleFunc("/api/console/export", s.handleExport)

	// Register web routes
	mux.HandleFunc("/", s.handleIndex)

	s.httpServer = &http.Server{
		Addr:         s.config.Host + ":" + s.config.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	addr := s.httpServer.Addr
	log.Printf("🔧 MetaBase Console listening on %s", addr)
	log.Printf("📊 Dashboard: http://localhost:%s", s.config.Port)

	return s.httpServer.ListenAndServe()
}

// Stop stops the console server
func (s *Server) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// Log logs an entry
func (s *Server) Log(level LogLevel, message string, data map[string]interface{}) {
	entry := LogEntry{
		ID:        generateLogID(),
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to logs (keep only max_logs)
	s.logs = append(s.logs, entry)
	if len(s.logs) > s.config.MaxLogs {
		s.logs = s.logs[1:]
	}

	// Update statistics
	s.updateStats(entry)

	log.Printf("[Console] %s: %s", level, message)
}

// LogRequest logs an HTTP request
func (s *Server) LogRequest(method, path string, status int, duration time.Duration, userID, ip, userAgent string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := LogEntry{
		ID:        generateLogID(),
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   fmt.Sprintf("%s %s %d", method, path, status),
		Method:    method,
		Status:    status,
		Duration:  duration,
		UserID:    userID,
		IP:        ip,
		UserAgent: userAgent,
		Data: map[string]interface{}{
			"path": path,
		},
	}

	// Add to logs
	s.logs = append(s.logs, entry)
	if len(s.logs) > s.config.MaxLogs {
		s.logs = s.logs[1:]
	}

	// Update statistics
	s.updateStats(entry)
}

// RecordMetric records a metric value
func (s *Server) RecordMetric(name string, value float64, tags map[string]string) {
	metric := MetricValue{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to metrics
	s.metrics[name] = append(s.metrics[name], metric)

	// Clean old metrics based on TTL
	cutoff := time.Now().Add(-s.config.MetricsTTL)
	if len(s.metrics[name]) > 10000 {
		var filtered []MetricValue
		for _, m := range s.metrics[name] {
			if m.Timestamp.After(cutoff) {
				filtered = append(filtered, m)
			}
		}
		s.metrics[name] = filtered
	}

	log.Printf("[Console] Metric: %s = %f", name, value)
}

// GetLogs returns logs with filtering
func (s *Server) GetLogs(level LogLevel, limit, offset int) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []LogEntry
	for _, log := range s.logs {
		if level == "" || log.Level == level {
			filtered = append(filtered, log)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply pagination
	if offset >= len(filtered) {
		return []LogEntry{}
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end]
}

// GetMetrics returns metrics for a given name and time range
func (s *Server) GetMetrics(name string, from, to time.Time) []MetricValue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.metrics[name]
	if !exists {
		return []MetricValue{}
	}

	var filtered []MetricValue
	for _, metric := range metrics {
		if (from.IsZero() || metric.Timestamp.After(from)) &&
		   (to.IsZero() || metric.Timestamp.Before(to)) {
			filtered = append(filtered, metric)
		}
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

// GetStats returns current statistics
func (s *Server) GetStats() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Copy stats
	statsCopy := *s.stats
	statsCopy.Throughput = make(map[string]float64)
	for k, v := range s.stats.Throughput {
		statsCopy.Throughput[k] = v
	}
	statsCopy.Custom = make(map[string]interface{})
	for k, v := range s.stats.Custom {
		statsCopy.Custom[k] = v
	}

	// Get storage stats
	if s.config.Storage != nil {
		statsCopy.Storage = s.config.Storage.Stats()
	}

	return &statsCopy
}

// Search searches through logs and metrics
func (s *Server) Search(query string, limit int) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string]interface{})
	var matchingLogs []LogEntry
	var matchingMetrics []MetricValue

	// Search logs
	for _, log := range s.logs {
		if s.containsQuery(log.Message, query) || s.matchesTags(log.Data, query) {
			matchingLogs = append(matchingLogs, log)
		}
	}

	// Search metrics
	for _, metrics := range s.metrics {
		for _, metric := range metrics {
			if s.containsQuery(metric.Name, query) || s.matchesTagsString(metric.Tags, query) {
				matchingMetrics = append(matchingMetrics, metric)
			}
		}
	}

	results["logs"] = matchingLogs
	if len(matchingLogs) > limit {
		results["logs"] = matchingLogs[:limit]
	}

	results["metrics"] = matchingMetrics
	if len(matchingMetrics) > limit {
		results["metrics"] = matchingMetrics[:limit]
	}

	results["total"] = len(matchingLogs) + len(matchingMetrics)

	return results
}

// HTTP Handlers
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 100
	}

	logs := s.GetLogs(LogLevel(level), limit, offset)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to time.Time
	if fromStr != "" {
		from, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr != "" {
		to, _ = time.Parse(time.RFC3339, toStr)
	}

	if name == "" {
		// Return all metric names
		s.mu.RLock()
		names := make([]string, 0, len(s.metrics))
		for name := range s.metrics {
			names = append(names, name)
		}
		s.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metrics": names,
		})
		return
	}

	metrics := s.GetMetrics(name, from, to)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metrics,
		"total":   len(metrics),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	results := s.Search(query, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)
		return
	}

	// Handle configuration updates
	if r.Method == "POST" {
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.config = &newConfig
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Configuration updated",
		})
	}
}

func (s *Server) handleLogDetails(w http.ResponseWriter, r *http.Request) {
	// Extract log ID from path
	logID := r.URL.Path[len("/api/console/logs/"):]

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, log := range s.logs {
		if log.ID == logID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(log)
			return
		}
	}

	http.Error(w, "Log not found", http.StatusNotFound)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=console-export.json")

		exportData := map[string]interface{}{
			"logs":     s.logs,
			"metrics":  s.metrics,
			"stats":    s.stats,
			"exported": time.Now(),
		}

		json.NewEncoder(w).Encode(exportData)

	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := s.generateDashboardHTML()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Helper methods
func (s *Server) initStats() {
	s.stats = &Stats{
		Throughput: make(map[string]float64),
		Custom:     make(map[string]interface{}),
		Latency: LatencyStats{
			P50: 0,
			P90: 0,
			P95: 0,
			P99: 0,
			Avg: 0,
		},
	}
}

func (s *Server) updateStats(entry LogEntry) {
	// Update QPS based on recent logs
	recentLogs := 0
	since := time.Now().Add(-time.Minute)
	for _, log := range s.logs {
		if log.Timestamp.After(since) && log.Method != "" {
			recentLogs++
		}
	}
	s.stats.QPS = float64(recentLogs) / 60.0

	// Update error rate
	errorCount := 0
	totalCount := 0
	for _, log := range s.logs {
		if log.Method != "" {
			totalCount++
			if log.Status >= 400 {
				errorCount++
			}
		}
	}
	if totalCount > 0 {
		s.stats.ErrorRate = float64(errorCount) / float64(totalCount) * 100.0
	}
}

func (s *Server) generateDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MetaBase Console</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .stat-value { font-size: 2em; font-weight: bold; color: #2563eb; }
        .stat-label { color: #6b7280; margin-top: 5px; }
        .logs-container { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .logs-header { padding: 20px; border-bottom: 1px solid #e5e7eb; }
        .logs-content { max-height: 400px; overflow-y: auto; }
        .log-entry { padding: 15px 20px; border-bottom: 1px solid #f3f4f6; font-family: monospace; font-size: 0.9em; }
        .log-level { padding: 2px 6px; border-radius: 3px; font-size: 0.8em; font-weight: bold; }
        .log-level.error { background: #fecaca; color: #dc2626; }
        .log-level.warn { background: #fed7aa; color: #ea580c; }
        .log-level.info { background: #dbeafe; color: #2563eb; }
        .log-level.debug { background: #e5e7eb; color: #6b7280; }
        .refresh-btn { background: #2563eb; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; }
        .refresh-btn:hover { background: #1d4ed8; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔧 MetaBase Console</h1>
            <p>系统监控、日志分析和性能统计</p>
            <button class="refresh-btn" onclick="location.reload()">🔄 刷新</button>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="qps">-</div>
                <div class="stat-label">QPS (每秒请求数)</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="error-rate">-</div>
                <div class="stat-label">错误率 (%)</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="avg-latency">-</div>
                <div class="stat-label">平均延迟</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="total-logs">-</div>
                <div class="stat-label">总日志数</div>
            </div>
        </div>

        <div class="logs-container">
            <div class="logs-header">
                <h2>📋 实时日志</h2>
            </div>
            <div class="logs-content" id="logs-content">
                <div class="log-entry">正在加载日志...</div>
            </div>
        </div>
    </div>

    <script>
        // Fetch stats
        fetch('/api/console/stats')
            .then(r => r.json())
            .then(data => {
                document.getElementById('qps').textContent = data.qps.toFixed(2);
                document.getElementById('error-rate').textContent = data.error_rate.toFixed(2);
                document.getElementById('avg-latency').textContent = data.latency.avg + 'ms';
            });

        // Fetch logs
        fetch('/api/console/logs?limit=50')
            .then(r => r.json())
            .then(data => {
                const container = document.getElementById('logs-content');
                document.getElementById('total-logs').textContent = data.total;

                if (data.logs && data.logs.length > 0) {
                    container.innerHTML = data.logs.map(log =>
                        '<div class="log-entry">' +
                        '<span class="log-level ' + log.level + '">' + log.level.toUpperCase() + '</span> ' +
                        '[' + new Date(log.timestamp).toLocaleString() + '] ' +
                        log.message +
                        '</div>'
                    ).join('');
                } else {
                    container.innerHTML = '<div class="log-entry">暂无日志</div>';
                }
            })
            .catch(err => {
                document.getElementById('logs-content').innerHTML =
                    '<div class="log-entry">加载日志失败: ' + err.message + '</div>';
            });

        // Auto refresh every 5 seconds
        setInterval(() => location.reload(), 5000);
    </script>
</body>
</html>`
}

func (s *Server) containsQuery(text, query string) bool {
	return len(text) >= len(query) &&
		   (text == query ||
		    len(text) > len(query) &&
		    s.containsSubstring(text, query))
}

func (s *Server) containsSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *Server) matchesTags(data map[string]interface{}, query string) bool {
	for _, v := range data {
		if str, ok := v.(string); ok && s.containsQuery(str, query) {
			return true
		}
	}
	return false
}

func (s *Server) matchesTagsString(tags map[string]string, query string) bool {
	for _, v := range tags {
		if s.containsQuery(v, query) {
			return true
		}
	}
	return false
}

func generateLogID() string {
	return fmt.Sprintf("log_%d", time.Now().UnixNano())
}