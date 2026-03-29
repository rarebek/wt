package wt

import (
	"net/http"
	"testing"
	"time"
)

func TestWithAddr(t *testing.T) {
	s := New(WithAddr(":9999"))
	if s.Addr() != ":9999" {
		t.Errorf("expected :9999, got %q", s.Addr())
	}
}

func TestWithIdleTimeout(t *testing.T) {
	s := New(WithIdleTimeout(5 * time.Minute))
	if s.idleTimeout != 5*time.Minute {
		t.Errorf("expected 5m, got %v", s.idleTimeout)
	}
}

func TestWithSelfSignedTLS(t *testing.T) {
	s := New(WithSelfSignedTLS())
	if s.autoTLS == nil {
		t.Error("expected autoTLS to be set")
	}
	if s.CertHash() == "" {
		t.Error("expected non-empty cert hash")
	}
}

func TestWithTLS(t *testing.T) {
	s := New(WithTLS("cert.pem", "key.pem"))
	if s.tlsCert != "cert.pem" || s.tlsKey != "key.pem" {
		t.Error("TLS file paths not set correctly")
	}
}

func TestWithCheckOrigin(t *testing.T) {
	s := New(WithCheckOrigin(func(r *http.Request) bool {
		return true
	}))
	if s.checkOrigin == nil {
		t.Error("expected checkOrigin to be set")
	}
}
