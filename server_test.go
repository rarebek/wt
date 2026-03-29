package wt

import "testing"

func TestNewServer(t *testing.T) {
	s := New(WithAddr(":4433"), WithSelfSignedTLS())
	if s.Addr() != ":4433" {
		t.Errorf("expected :4433, got %q", s.Addr())
	}
	if s.CertHash() == "" {
		t.Error("expected non-empty cert hash for self-signed")
	}
	if s.SessionCount() != 0 {
		t.Errorf("expected 0 sessions, got %d", s.SessionCount())
	}
}

func TestNewServerDefaults(t *testing.T) {
	s := New()
	if s.Addr() != ":4433" {
		t.Errorf("expected default :4433, got %q", s.Addr())
	}
}

func TestServerVersion(t *testing.T) {
	if Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestServerInfoFromServer(t *testing.T) {
	info := ServerInfo()
	if info["version"] != Version {
		t.Error("expected version match")
	}
}
