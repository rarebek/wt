package wt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAltSvcHeader(t *testing.T) {
	header := AltSvcHeader(4433)
	expected := `h3=":4433"; ma=86400`
	if header != expected {
		t.Errorf("expected %q, got %q", expected, header)
	}
}

func TestSetAltSvcHeader(t *testing.T) {
	w := httptest.NewRecorder()
	SetAltSvcHeader(w, 443)

	got := w.Header().Get("Alt-Svc")
	if got != `h3=":443"; ma=86400` {
		t.Errorf("expected Alt-Svc header, got %q", got)
	}
}

func TestAltSvcMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := AltSvcMiddleware(4433)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	got := w.Header().Get("Alt-Svc")
	if got == "" {
		t.Error("expected Alt-Svc header to be set")
	}
}
