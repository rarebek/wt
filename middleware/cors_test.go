package middleware

import "testing"

func TestCORSConfigAllowAll(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
	}

	// With wildcard, any origin should pass
	if len(config.AllowedOrigins) != 1 || config.AllowedOrigins[0] != "*" {
		t.Error("expected wildcard origin")
	}
}

func TestCORSConfigSpecific(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
	}

	allowed := make(map[string]bool)
	for _, o := range config.AllowedOrigins {
		allowed[o] = true
	}

	if !allowed["https://example.com"] {
		t.Error("expected example.com to be allowed")
	}
	if !allowed["https://app.example.com"] {
		t.Error("expected app.example.com to be allowed")
	}
	if allowed["https://evil.com"] {
		t.Error("evil.com should not be allowed")
	}
}
