package wt

import (
	"golang.org/x/crypto/acme/autocert"
)

// WithAutoCert configures automatic TLS certificate management via Let's Encrypt.
// Certificates are automatically obtained and renewed.
//
// Requirements:
//   - The server must be publicly accessible on port 443
//   - DNS must point to this server
//   - A cache directory stores certificates (e.g., "/var/cache/certs")
//
// Note: ACME validation uses TLS-ALPN-01 challenge, which requires port 443 TCP.
// The WebTransport server itself runs on UDP, so you need both:
//   - TCP port 443 for ACME challenges (handled by autocert)
//   - UDP port 443 for QUIC/WebTransport (handled by the framework)
//
// Usage:
//
//	server := wt.New(
//	    wt.WithAddr(":443"),
//	    wt.WithAutoCert("example.com", "/var/cache/certs"),
//	)
func WithAutoCert(domain string, cacheDir string) Option {
	return WithAutoCertMulti([]string{domain}, cacheDir)
}

// WithAutoCertMulti is like WithAutoCert but supports multiple domains.
func WithAutoCertMulti(domains []string, cacheDir string) Option {
	return func(s *Server) {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domains...),
			Cache:      autocert.DirCache(cacheDir),
		}

		s.tlsCert = ""
		s.tlsKey = ""
		s.autoTLS = nil
		s.autocertManager = m
	}
}

