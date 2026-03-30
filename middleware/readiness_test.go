package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessProbe(t *testing.T) {
	r := NewReadiness()

	// Initially not ready
	if r.IsReady() {
		t.Error("should not be ready initially")
	}

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	r.ReadinessHandler().ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	// Mark ready
	r.SetReady(true)
	w = httptest.NewRecorder()
	r.ReadinessHandler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLivenessProbe(t *testing.T) {
	r := NewReadiness()

	// Healthy by default
	if !r.IsHealthy() {
		t.Error("should be healthy by default")
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.LivenessHandler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Mark unhealthy
	r.SetHealthy(false)
	w = httptest.NewRecorder()
	r.LivenessHandler().ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
