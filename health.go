package wt

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthCheck provides an HTTP health check endpoint that reports server status.
// Serve this alongside your WebTransport server on an HTTP port for load balancers
// and monitoring systems.
type HealthCheck struct {
	server    *Server
	startTime time.Time
}

// NewHealthCheck creates a health check handler for the given server.
func NewHealthCheck(s *Server) *HealthCheck {
	return &HealthCheck{
		server:    s,
		startTime: time.Now(),
	}
}

// HealthResponse is the JSON response from the health check endpoint.
type HealthResponse struct {
	Status         string `json:"status"`
	ActiveSessions int    `json:"active_sessions"`
	Uptime         string `json:"uptime"`
	Transport      string `json:"transport"`
}

// ServeHTTP implements http.Handler for health checks.
func (h *HealthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:         "ok",
		ActiveSessions: h.server.Sessions().Count(),
		Uptime:         time.Since(h.startTime).Truncate(time.Second).String(),
		Transport:      "webtransport",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Handler returns an http.Handler for health checks.
func (h *HealthCheck) Handler() http.Handler {
	return h
}
