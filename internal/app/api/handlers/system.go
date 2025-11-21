package handlers

import ("encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap")

// SystemHandler handles system-related requests
type SystemHandler struct {
	logger *zap.Logger
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(logger *zap.Logger) *SystemHandler {
	return &SystemHandler{
		logger: logger,
	}
}

// Health handles health check requests
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"service":   "metabase-api",
		"uptime":    "0h", // TODO: calculate actual uptime
	}

	h.writeJSON(w, health)
}

// Ping handles ping requests
func (h *SystemHandler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

// Version handles version requests
func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":     "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339),
		"go_version":  "go1.25.3",
		"git_commit":  "unknown", // TODO: get actual git commit
		"service":     "metabase-api",
		"environment": "development",
	}

	h.writeJSON(w, version)
}

// Helper methods
func (h *SystemHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}