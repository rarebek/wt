package wt

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CertRotator watches TLS certificate files and reloads them without server restart.
// QUIC connections established before rotation continue using the old cert.
// New connections use the new cert.
//
// Usage:
//
//	rotator := wt.NewCertRotator("cert.pem", "key.pem",
//	    wt.WithRotationInterval(1*time.Hour),
//	)
//	server := wt.New(
//	    wt.WithAddr(":443"),
//	    wt.WithCertRotator(rotator),
//	)
type CertRotator struct {
	mu       sync.RWMutex
	cert     *tls.Certificate
	certFile string
	keyFile  string
	interval time.Duration
	logger   *slog.Logger
	done     chan struct{}
}

// RotatorOption configures the CertRotator.
type RotatorOption func(*CertRotator)

// WithRotationInterval sets how often to check for new certificates (default: 1 hour).
func WithRotationInterval(d time.Duration) RotatorOption {
	return func(cr *CertRotator) { cr.interval = d }
}

// WithRotationLogger sets the logger for rotation events.
func WithRotationLogger(logger *slog.Logger) RotatorOption {
	return func(cr *CertRotator) { cr.logger = logger }
}

// NewCertRotator creates a certificate rotator.
func NewCertRotator(certFile, keyFile string, opts ...RotatorOption) (*CertRotator, error) {
	cr := &CertRotator{
		certFile: certFile,
		keyFile:  keyFile,
		interval: 1 * time.Hour,
		logger:   slog.Default(),
		done:     make(chan struct{}),
	}
	for _, opt := range opts {
		opt(cr)
	}

	// Load initial cert
	if err := cr.reload(); err != nil {
		return nil, fmt.Errorf("wt: initial cert load: %w", err)
	}

	go cr.watchLoop()
	return cr, nil
}

// GetCertificate returns the current certificate. Implements tls.Config.GetCertificate.
func (cr *CertRotator) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	cr.mu.RLock()
	cert := cr.cert
	cr.mu.RUnlock()
	return cert, nil
}

// TLSConfig returns a tls.Config that uses the rotator for certificates.
func (cr *CertRotator) TLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: cr.GetCertificate,
		NextProtos:     []string{"h3"},
	}
}

// Stop stops the certificate watcher.
func (cr *CertRotator) Stop() {
	select {
	case <-cr.done:
	default:
		close(cr.done)
	}
}

func (cr *CertRotator) reload() error {
	cert, err := tls.LoadX509KeyPair(cr.certFile, cr.keyFile)
	if err != nil {
		return err
	}

	cr.mu.Lock()
	cr.cert = &cert
	cr.mu.Unlock()

	cr.logger.Info("certificate reloaded",
		"cert_file", cr.certFile,
		"key_file", cr.keyFile,
	)
	return nil
}

func (cr *CertRotator) watchLoop() {
	ticker := time.NewTicker(cr.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cr.reload(); err != nil {
				cr.logger.Error("certificate reload failed",
					"error", err,
					"cert_file", cr.certFile,
				)
				// Continue using old cert
			}
		case <-cr.done:
			return
		}
	}
}

// WithCertRotator configures the server to use a CertRotator for TLS.
func WithCertRotator(cr *CertRotator) Option {
	return func(s *Server) {
		s.certRotator = cr
	}
}
