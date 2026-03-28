package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusHandler(t *testing.T) {
	pm := NewPrometheusMetrics()

	// Simulate some sessions
	pm.activeSessions.Add(5)
	pm.totalSessions.Add(10)
	pm.sessionDurations.Add(5_000_000_000) // 5 seconds

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	pm.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "wt_active_sessions 5") {
		t.Error("expected wt_active_sessions 5")
	}
	if !strings.Contains(body, "wt_sessions_total 10") {
		t.Error("expected wt_sessions_total 10")
	}
	if !strings.Contains(body, "wt_session_duration_seconds_total 5.") {
		t.Error("expected total duration ~5 seconds")
	}
	if !strings.Contains(body, "wt_session_duration_seconds_avg 0.5") {
		t.Error("expected avg duration ~0.5 seconds")
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected text/plain content type, got %q", contentType)
	}
}

func TestPrometheusEmptyMetrics(t *testing.T) {
	pm := NewPrometheusMetrics()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	pm.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, "wt_active_sessions 0") {
		t.Error("expected wt_active_sessions 0")
	}
	if !strings.Contains(body, "wt_sessions_total 0") {
		t.Error("expected wt_sessions_total 0")
	}
	// No avg when total is 0
	if strings.Contains(body, "wt_session_duration_seconds_avg") {
		t.Error("should not have avg when no sessions")
	}
}
