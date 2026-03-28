package wt

import "testing"

func TestPreflightNoTLS(t *testing.T) {
	server := New(WithAddr(":0"))
	issues := server.Preflight()

	found := false
	for _, issue := range issues {
		if issue == "no TLS configuration: use WithTLS(), WithSelfSignedTLS(), WithAutoCert(), or WithCertRotator()" {
			found = true
		}
	}
	if !found {
		t.Error("expected TLS warning")
	}
}

func TestPreflightWithSelfSigned(t *testing.T) {
	server := New(WithAddr("127.0.0.1:0"), WithSelfSignedTLS())
	result := server.PreflightCheck()

	if !result.Ready {
		t.Errorf("expected ready, got issues: %v", result.Issues)
	}
}

func TestPreflightBadCert(t *testing.T) {
	server := New(WithAddr(":0"), WithTLS("/nonexistent/cert.pem", "/nonexistent/key.pem"))
	issues := server.Preflight()

	found := false
	for _, issue := range issues {
		if len(issue) > 10 { // has a cert error message
			found = true
		}
	}
	if !found {
		t.Error("expected cert error")
	}
}
