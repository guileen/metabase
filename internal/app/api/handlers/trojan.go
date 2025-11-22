package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/guileen/metabase/internal/app/trojan"
	trojanpkg "github.com/guileen/metabase/pkg/trojan"
	"go.uber.org/zap"
)

// TrojanHandler handles Trojan VPN API requests
type TrojanHandler struct {
	manager *trojan.Manager
	logger  *zap.Logger
}

// NewTrojanHandler creates a new Trojan handler
func NewTrojanHandler(manager *trojan.Manager, logger *zap.Logger) *TrojanHandler {
	return &TrojanHandler{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes registers Trojan API routes
func (h *TrojanHandler) RegisterRoutes(r chi.Router) {
	r.Route("/trojan", func(r chi.Router) {
		// Service management
		r.Get("/status", h.GetStatus)
		r.Post("/start", h.StartService)
		r.Post("/stop", h.StopService)
		r.Post("/restart", h.RestartService)

		// Configuration
		r.Get("/config", h.GetConfig)
		r.Put("/config", h.UpdateConfig)

		// Client management
		r.Get("/clients", h.ListClients)
		r.Post("/clients", h.AddClient)
		r.Get("/clients/{id}", h.GetClient)
		r.Put("/clients/{id}", h.UpdateClient)
		r.Delete("/clients/{id}", h.RemoveClient)
		r.Get("/clients/{id}/stats", h.GetClientStats)
		r.Get("/clients/{id}/config", h.GetClientConfig)

		// Statistics and monitoring
		r.Get("/stats", h.GetStats)
		r.Get("/connections", h.GetConnections)
		r.Get("/certificate", h.GetCertificateInfo)

		// Utilities
		r.Post("/generate-client-password", h.GenerateClientPassword)
	})
}

// Request/Response types

type StartRequest struct {
	Config *trojanpkg.TrojanConfig `json:"config,omitempty"`
}

type ConfigRequest struct {
	Config *trojanpkg.TrojanConfig `json:"config"`
}

type ClientRequest struct {
	Client *trojanpkg.ClientInfo `json:"client"`
}

type ClientUpdateRequest struct {
	Name        string     `json:"name,omitempty"`
	Status      string     `json:"status,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	DataLimit   int64      `json:"data_limit,omitempty"`
	IPWhitelist []string   `json:"ip_whitelist,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

// Service management handlers

func (h *TrojanHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetStatus()
	respondWithJSON(w, http.StatusOK, status)
}

func (h *TrojanHandler) StartService(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update config if provided
	if req.Config != nil {
		if err := h.manager.UpdateConfig(req.Config); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := h.manager.Start(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Trojan service started successfully",
	})
}

func (h *TrojanHandler) StopService(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.Stop(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Trojan service stopped successfully",
	})
}

func (h *TrojanHandler) RestartService(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update config if provided
	if req.Config != nil {
		if err := h.manager.UpdateConfig(req.Config); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := h.manager.Restart(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Trojan service restarted successfully",
	})
}

// Configuration handlers

func (h *TrojanHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetStatus()
	respondWithJSON(w, http.StatusOK, status.Config)
}

func (h *TrojanHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.manager.UpdateConfig(req.Config); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Configuration updated successfully",
	})
}

// Client management handlers

func (h *TrojanHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.manager.ListClients()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"clients": clients,
		"total":   len(clients),
	})
}

func (h *TrojanHandler) AddClient(w http.ResponseWriter, r *http.Request) {
	var req ClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate ID if not provided
	if req.Client.ID == "" {
		req.Client.ID = generateClientID()
	}

	// Set created time if not provided
	if req.Client.CreatedAt.IsZero() {
		req.Client.CreatedAt = time.Now()
	}

	if err := h.manager.AddClient(req.Client); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Client added successfully",
		"client":  req.Client,
	})
}

func (h *TrojanHandler) GetClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	client, err := h.manager.GetClient(clientID)
	if err != nil {
		if err == trojanpkg.ErrClientNotFound {
			respondWithError(w, http.StatusNotFound, "Client not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, client)
}

func (h *TrojanHandler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	var req ClientUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing client
	client, err := h.manager.GetClient(clientID)
	if err != nil {
		if err == trojanpkg.ErrClientNotFound {
			respondWithError(w, http.StatusNotFound, "Client not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Update fields
	if req.Name != "" {
		client.Name = req.Name
	}
	if req.Status != "" {
		client.Status = req.Status
	}
	if req.ExpiresAt != nil {
		client.ExpiresAt = req.ExpiresAt
	}
	if req.DataLimit > 0 {
		client.DataLimit = req.DataLimit
	}
	if req.IPWhitelist != nil {
		client.IPWhitelist = req.IPWhitelist
	}
	if req.Tags != nil {
		client.Tags = req.Tags
	}

	// Remove and re-add client (simple update mechanism)
	if err := h.manager.RemoveClient(clientID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.manager.AddClient(client); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Client updated successfully",
		"client":  client,
	})
}

func (h *TrojanHandler) RemoveClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	if err := h.manager.RemoveClient(clientID); err != nil {
		if err == trojanpkg.ErrClientNotFound {
			respondWithError(w, http.StatusNotFound, "Client not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Client removed successfully",
	})
}

func (h *TrojanHandler) GetClientStats(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	// Get all client stats and filter by ID
	allStats, err := h.manager.GetClientStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, stats := range allStats {
		if stats.ClientID == clientID {
			respondWithJSON(w, http.StatusOK, stats)
			return
		}
	}

	respondWithError(w, http.StatusNotFound, "Client stats not found")
}

func (h *TrojanHandler) GetClientConfig(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "id")

	config, err := h.manager.GenerateClientConfig(clientID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, config)
}

// Statistics and monitoring handlers

func (h *TrojanHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.manager.GetClientStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"client_stats":  stats,
		"total_clients": len(stats),
	})
}

func (h *TrojanHandler) GetConnections(w http.ResponseWriter, r *http.Request) {
	status := h.manager.GetStatus()

	if status.Connections == nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"connections": []*trojanpkg.Connection{},
			"total":       0,
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"connections": status.Connections,
		"total":       len(status.Connections),
	})
}

func (h *TrojanHandler) GetCertificateInfo(w http.ResponseWriter, r *http.Request) {
	// This would need to be implemented in the manager
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Certificate info not yet implemented",
	})
}

// Utility handlers

func (h *TrojanHandler) GenerateClientPassword(w http.ResponseWriter, r *http.Request) {
	// Generate a secure password
	password := generateSecurePassword()

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"password": password,
	})
}

// Helper functions

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

func generateSecurePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 16)
	for i := range password {
		password[i] = charset[i%len(charset)]
	}
	return string(password)
}

// Common response helpers

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]interface{}{
		"error": message,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to encode response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
