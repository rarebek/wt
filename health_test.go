package wt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	hc := NewHealthCheck(server)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	hc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Transport != "webtransport" {
		t.Errorf("expected transport 'webtransport', got %q", resp.Transport)
	}
	if resp.ActiveSessions != 0 {
		t.Errorf("expected 0 active sessions, got %d", resp.ActiveSessions)
	}
	if resp.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}
