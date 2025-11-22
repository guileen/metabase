package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/config"
	"github.com/guileen/metabase/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Global metrics instance
var (
	globalMetrics *Metrics
	once          sync.Once
)

// Metrics provides a comprehensive metrics collection system
type Metrics struct {
	config     *config.MetricsConfig
	registry   *prometheus.Registry
	logger     *log.Logger
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mu         sync.RWMutex
}

// MetricConfig holds configuration for individual metrics
type MetricConfig struct {
	Name        string              `yaml:"name"`
	Help        string              `yaml:"help"`
	Type        string              `yaml:"type"` // counter, gauge, histogram, summary
	Labels      []string            `yaml:"labels"`
	Buckets     []float64           `yaml:"buckets"`    // For histograms
	Objectives  map[float64]float64 `yaml:"objectives"` // For summaries
	ConstLabels map[string]string   `yaml:"const_labels"`
}

// Predefined metric configurations
var DefaultMetrics = []MetricConfig{
	{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
		Type: "counter",
		Labels: []string{
			"method",
			"path",
			"status_code",
			"component",
		},
	},
	{
		Name: "http_request_duration_seconds",
		Help: "HTTP request duration in seconds",
		Type: "histogram",
		Labels: []string{
			"method",
			"path",
			"status_code",
			"component",
		},
		Buckets: prometheus.DefBuckets,
	},
	{
		Name: "http_response_size_bytes",
		Help: "HTTP response size in bytes",
		Type: "histogram",
		Labels: []string{
			"method",
			"path",
			"status_code",
			"component",
		},
		Buckets: prometheus.ExponentialBuckets(100, 2, 10),
	},
	{
		Name: "active_connections",
		Help: "Number of active connections",
		Type: "gauge",
		Labels: []string{
			"component",
			"connection_type",
		},
	},
	{
		Name: "log_messages_total",
		Help: "Total number of log messages",
		Type: "counter",
		Labels: []string{
			"level",
			"component",
		},
	},
	{
		Name: "errors_total",
		Help: "Total number of errors",
		Type: "counter",
		Labels: []string{
			"error_type",
			"component",
			"severity",
		},
	},
	{
		Name: "database_connections_active",
		Help: "Number of active database connections",
		Type: "gauge",
		Labels: []string{
			"database",
			"component",
		},
	},
	{
		Name: "database_query_duration_seconds",
		Help: "Database query duration in seconds",
		Type: "histogram",
		Labels: []string{
			"database",
			"operation",
			"table",
			"component",
		},
		Buckets: []float64{0.001, 0.01, 0.1, 1.0, 10.0},
	},
	{
		Name: "cache_operations_total",
		Help: "Total number of cache operations",
		Type: "counter",
		Labels: []string{
			"operation",
			"cache",
			"result",
			"component",
		},
	},
	{
		Name: "system_memory_bytes",
		Help: "System memory usage in bytes",
		Type: "gauge",
		Labels: []string{
			"type", // used, free, cached
			"component",
		},
	},
}

// NewMetrics creates a new metrics instance
func NewMetrics(cfg *config.MetricsConfig) (*Metrics, error) {
	if cfg == nil {
		cfg = &config.MetricsConfig{
			Enabled:   true,
			Port:      9090,
			Path:      "/metrics",
			Namespace: "metabase",
			Subsystem: "server",
		}
	}

	metrics := &Metrics{
		config:     cfg,
		registry:   prometheus.NewRegistry(),
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
	}

	// Set logger
	metrics.logger = log.Get()

	// Register default collectors
	metrics.registry.MustRegister(prometheus.NewGoCollector())
	metrics.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// Initialize default metrics
	for _, metricConfig := range DefaultMetrics {
		if err := metrics.RegisterMetric(metricConfig); err != nil {
			metrics.logger.WithComponent("metrics").Warn("Failed to register metric",
				"name", metricConfig.Name,
				"error", err,
			)
		}
	}

	return metrics, nil
}

// Initialize initializes the global metrics instance
func Initialize(cfg *config.MetricsConfig) error {
	var err error
	once.Do(func() {
		globalMetrics, err = NewMetrics(cfg)
	})
	return err
}

// Get returns the global metrics instance
func Get() *Metrics {
	once.Do(func() {
		if globalMetrics == nil {
			globalMetrics, _ = NewMetrics(nil)
		}
	})
	return globalMetrics
}

// MustGet returns the global metrics instance and panics if not initialized
func MustGet() *Metrics {
	metrics := Get()
	if metrics == nil {
		panic("Metrics not initialized")
	}
	return metrics
}

// RegisterMetric registers a new metric
func (m *Metrics) RegisterMetric(config MetricConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	namespace := m.config.Namespace
	subsystem := m.config.Subsystem
	name := prometheus.BuildFQName(namespace, subsystem, config.Name)

	constLabels := make(map[string]string)
	for k, v := range config.ConstLabels {
		constLabels[k] = v
	}

	switch config.Type {
	case "counter":
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        name,
				Help:        config.Help,
				ConstLabels: constLabels,
			},
			config.Labels,
		)
		if err := m.registry.Register(counter); err != nil {
			return fmt.Errorf("failed to register counter %s: %w", name, err)
		}
		m.counters[config.Name] = counter

	case "gauge":
		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        name,
				Help:        config.Help,
				ConstLabels: constLabels,
			},
			config.Labels,
		)
		if err := m.registry.Register(gauge); err != nil {
			return fmt.Errorf("failed to register gauge %s: %w", name, err)
		}
		m.gauges[config.Name] = gauge

	case "histogram":
		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        name,
				Help:        config.Help,
				Buckets:     config.Buckets,
				ConstLabels: constLabels,
			},
			config.Labels,
		)
		if err := m.registry.Register(histogram); err != nil {
			return fmt.Errorf("failed to register histogram %s: %w", name, err)
		}
		m.histograms[config.Name] = histogram

	case "summary":
		summary := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:        name,
				Help:        config.Help,
				Objectives:  config.Objectives,
				ConstLabels: constLabels,
			},
			config.Labels,
		)
		if err := m.registry.Register(summary); err != nil {
			return fmt.Errorf("failed to register summary %s: %w", name, err)
		}
		m.summaries[config.Name] = summary

	default:
		return fmt.Errorf("unknown metric type: %s", config.Type)
	}

	return nil
}

// Counter increments a counter metric
func (m *Metrics) Counter(name string, labels prometheus.Labels) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if counter, exists := m.counters[name]; exists {
		counter.With(labels).Inc()
	} else {
		m.logger.Warn("Counter not found", "name", name)
	}
}

// Gauge sets a gauge metric value
func (m *Metrics) Gauge(name string, value float64, labels prometheus.Labels) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if gauge, exists := m.gauges[name]; exists {
		gauge.With(labels).Set(value)
	} else {
		m.logger.Warn("Gauge not found", "name", name)
	}
}

// Histogram observes a histogram metric value
func (m *Metrics) Histogram(name string, value float64, labels prometheus.Labels) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if histogram, exists := m.histograms[name]; exists {
		histogram.With(labels).Observe(value)
	} else {
		m.logger.Warn("Histogram not found", "name", name)
	}
}

// Summary observes a summary metric value
func (m *Metrics) Summary(name string, value float64, labels prometheus.Labels) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if summary, exists := m.summaries[name]; exists {
		summary.With(labels).Observe(value)
	} else {
		m.logger.Warn("Summary not found", "name", name)
	}
}

// IncrementCounter is a convenience function for incrementing counters
func (m *Metrics) IncrementCounter(name string, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if counter, exists := m.counters[name]; exists {
		counter.WithLabelValues(labelValues...).Inc()
	} else {
		m.logger.Warn("Counter not found", "name", name)
	}
}

// SetGauge is a convenience function for setting gauge values
func (m *Metrics) SetGauge(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if gauge, exists := m.gauges[name]; exists {
		gauge.WithLabelValues(labelValues...).Set(value)
	} else {
		m.logger.Warn("Gauge not found", "name", name)
	}
}

// ObserveHistogram is a convenience function for observing histogram values
func (m *Metrics) ObserveHistogram(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if histogram, exists := m.histograms[name]; exists {
		histogram.WithLabelValues(labelValues...).Observe(value)
	} else {
		m.logger.Warn("Histogram not found", "name", name)
	}
}

// ObserveSummary is a convenience function for observing summary values
func (m *Metrics) ObserveSummary(name string, value float64, labelValues ...string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if summary, exists := m.summaries[name]; exists {
		summary.WithLabelValues(labelValues...).Observe(value)
	} else {
		m.logger.Warn("Summary not found", "name", name)
	}
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, responseSize int64, component string) {
	labels := prometheus.Labels{
		"method":      method,
		"path":        path,
		"status_code": fmt.Sprintf("%d", statusCode),
		"component":   component,
	}

	// Increment request counter
	m.Counter("http_requests_total", labels)

	// Record request duration
	m.Histogram("http_request_duration_seconds", duration.Seconds(), labels)

	// Record response size
	if responseSize > 0 {
		m.Histogram("http_response_size_bytes", float64(responseSize), labels)
	}
}

// RecordLogMessage records log message metrics
func (m *Metrics) RecordLogMessage(level string, component string) {
	labels := prometheus.Labels{
		"level":     level,
		"component": component,
	}
	m.Counter("log_messages_total", labels)
}

// RecordError records error metrics
func (m *Metrics) RecordError(errorType, component, severity string) {
	labels := prometheus.Labels{
		"error_type": errorType,
		"component":  component,
		"severity":   severity,
	}
	m.Counter("errors_total", labels)
}

// RecordDatabaseQuery records database query metrics
func (m *Metrics) RecordDatabaseQuery(database, operation, table, component string, duration time.Duration) {
	labels := prometheus.Labels{
		"database":  database,
		"operation": operation,
		"table":     table,
		"component": component,
	}
	m.Histogram("database_query_duration_seconds", duration.Seconds(), labels)
}

// RecordCacheOperation records cache operation metrics
func (m *Metrics) RecordCacheOperation(operation, cache, result, component string) {
	labels := prometheus.Labels{
		"operation": operation,
		"cache":     cache,
		"result":    result,
		"component": component,
	}
	m.Counter("cache_operations_total", labels)
}

// GetRegistry returns the Prometheus registry
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// StartMetricsServer starts the metrics HTTP server
func (m *Metrics) StartMetricsServer(ctx context.Context) error {
	if !m.config.Enabled {
		m.logger.Info("Metrics server disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(m.config.Path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.Port),
		Handler: mux,
	}

	m.logger.Info("Starting metrics server",
		"port", m.config.Port,
		"path", m.config.Path,
	)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server failed", "error", err)
		}
	}()

	// Handle shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		m.logger.Info("Shutting down metrics server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			m.logger.Error("Metrics server shutdown failed", "error", err)
		}
	}()

	return nil
}

// GetMetricNames returns all registered metric names
func (m *Metrics) GetMetricNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string

	for name := range m.counters {
		names = append(names, name)
	}
	for name := range m.gauges {
		names = append(names, name)
	}
	for name := range m.histograms {
		names = append(names, name)
	}
	for name := range m.summaries {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetMetricInfo returns information about all registered metrics
func (m *Metrics) GetMetricInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make(map[string]interface{})

	for name, counter := range m.counters {
		info[name] = map[string]interface{}{
			"type":   "counter",
			"help":   getMetricDesc(counter),
			"labels": getMetricLabels(counter),
		}
	}

	for name, gauge := range m.gauges {
		info[name] = map[string]interface{}{
			"type":   "gauge",
			"help":   getMetricDesc(gauge),
			"labels": getMetricLabels(gauge),
		}
	}

	for name, histogram := range m.histograms {
		info[name] = map[string]interface{}{
			"type":   "histogram",
			"help":   getMetricDesc(histogram),
			"labels": getMetricLabels(histogram),
		}
	}

	for name, summary := range m.summaries {
		info[name] = map[string]interface{}{
			"type":   "summary",
			"help":   getMetricDesc(summary),
			"labels": getMetricLabels(summary),
		}
	}

	return info
}

// Global convenience functions

func Counter(name string, labels prometheus.Labels) {
	Get().Counter(name, labels)
}

func Gauge(name string, value float64, labels prometheus.Labels) {
	Get().Gauge(name, value, labels)
}

func Histogram(name string, value float64, labels prometheus.Labels) {
	Get().Histogram(name, value, labels)
}

func Summary(name string, value float64, labels prometheus.Labels) {
	Get().Summary(name, value, labels)
}

func IncrementCounter(name string, labelValues ...string) {
	Get().IncrementCounter(name, labelValues...)
}

func SetGauge(name string, value float64, labelValues ...string) {
	Get().SetGauge(name, value, labelValues...)
}

func ObserveHistogram(name string, value float64, labelValues ...string) {
	Get().ObserveHistogram(name, value, labelValues...)
}

func ObserveSummary(name string, value float64, labelValues ...string) {
	Get().ObserveSummary(name, value, labelValues...)
}

func RecordHTTPRequest(method, path string, statusCode int, duration time.Duration, responseSize int64, component string) {
	Get().RecordHTTPRequest(method, path, statusCode, duration, responseSize, component)
}

func RecordLogMessage(level string, component string) {
	Get().RecordLogMessage(level, component)
}

func RecordError(errorType, component, severity string) {
	Get().RecordError(errorType, component, severity)
}

func RecordDatabaseQuery(database, operation, table, component string, duration time.Duration) {
	Get().RecordDatabaseQuery(database, operation, table, component, duration)
}

func RecordCacheOperation(operation, cache, result, component string) {
	Get().RecordCacheOperation(operation, cache, result, component)
}

// Helper functions

func getMetricDesc(metric interface{}) string {
	// This is a simplified implementation that returns the metric name
	// In a real implementation, you'd extract the description from the metric descriptor
	switch v := metric.(type) {
	case *prometheus.CounterVec:
		if v != nil {
			return "Counter metric"
		}
	case *prometheus.GaugeVec:
		if v != nil {
			return "Gauge metric"
		}
	case *prometheus.HistogramVec:
		if v != nil {
			return "Histogram metric"
		}
	case *prometheus.SummaryVec:
		if v != nil {
			return "Summary metric"
		}
	}
	return "Unknown metric"
}

func getMetricLabels(metric interface{}) []string {
	// This is a simplified implementation
	// In a real implementation, you'd extract label names from the metric descriptor
	return []string{}
}
