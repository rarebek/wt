package wt

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

// PreflightCheck verifies the server configuration before starting.
// Returns a list of issues found. Empty list = ready to start.
//
// Usage:
//
//	server := wt.New(...)
//	if issues := server.Preflight(); len(issues) > 0 {
//	    for _, issue := range issues {
//	        log.Printf("WARN: %s", issue)
//	    }
//	}
func (s *Server) Preflight() []string {
	var issues []string

	// Check address format
	host, port, err := net.SplitHostPort(s.addr)
	if err != nil {
		issues = append(issues, fmt.Sprintf("invalid address %q: %v", s.addr, err))
		return issues // can't continue without valid address
	}
	_ = host

	// Check port availability
	conn, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		issues = append(issues, fmt.Sprintf("port %s unavailable: %v", port, err))
	} else {
		conn.Close()
	}

	// Check TLS configuration
	hasTLS := false
	if s.autoTLS != nil {
		hasTLS = true
	}
	if s.autocertManager != nil {
		hasTLS = true
	}
	if s.certRotator != nil {
		hasTLS = true
	}
	if s.tlsCert != "" && s.tlsKey != "" {
		hasTLS = true
		// Try to load the certificate files
		_, err := tls.LoadX509KeyPair(s.tlsCert, s.tlsKey)
		if err != nil {
			issues = append(issues, fmt.Sprintf("TLS cert error: %v", err))
		}
	}

	if !hasTLS {
		issues = append(issues, "no TLS configuration: use WithTLS(), WithSelfSignedTLS(), WithAutoCert(), or WithCertRotator()")
	}

	// Check for common misconfigurations
	if port == "443" && s.autoTLS == nil && s.autocertManager == nil {
		issues = append(issues, "port 443 usually requires proper TLS certificates (not self-signed)")
	}

	// Warn about self-signed certs in non-dev settings
	if s.autoTLS != nil && !strings.Contains(s.addr, "localhost") && !strings.Contains(s.addr, "127.0.0.1") {
		issues = append(issues, "self-signed TLS is for development only — use WithTLS() or WithAutoCert() in production")
	}

	return issues
}

// PreflightResult holds the result of a preflight check.
type PreflightResult struct {
	Ready  bool
	Issues []string
}

// PreflightCheck runs the preflight check and returns a structured result.
func (s *Server) PreflightCheck() PreflightResult {
	issues := s.Preflight()
	return PreflightResult{
		Ready:  len(issues) == 0,
		Issues: issues,
	}
}
