package middleware

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Readiness provides Kubernetes-style readiness and liveness probes.
// Set ready=false during startup and shutdown to stop receiving traffic.
type Readiness struct {
	ready   atomic.Bool
	healthy atomic.Bool
}

// NewReadiness creates a readiness controller, initially not ready.
func NewReadiness() *Readiness {
	r := &Readiness{}
	r.healthy.Store(true) // healthy by default
	return r
}

// SetReady marks the service as ready to receive traffic.
func (r *Readiness) SetReady(ready bool) { r.ready.Store(ready) }

// SetHealthy marks the service as healthy.
func (r *Readiness) SetHealthy(healthy bool) { r.healthy.Store(healthy) }

// IsReady returns whether the service is ready.
func (r *Readiness) IsReady() bool { return r.ready.Load() }

// IsHealthy returns whether the service is healthy.
func (r *Readiness) IsHealthy() bool { return r.healthy.Load() }

// ReadinessHandler returns an HTTP handler for /readyz.
func (r *Readiness) ReadinessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.IsReady() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		}
	})
}

// LivenessHandler returns an HTTP handler for /healthz.
func (r *Readiness) LivenessHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.IsHealthy() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
		}
	})
}
