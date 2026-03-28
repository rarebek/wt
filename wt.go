// Package wt provides a high-level framework for building WebTransport applications in Go.
// It sits on top of quic-go/webtransport-go and provides routing, middleware,
// session management, and codec support.
package wt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// Server is the main WebTransport framework server.
type Server struct {
	wt     *webtransport.Server
	router *Router
	mw     []MiddlewareFunc
	mu     sync.RWMutex

	sessions *SessionStore

	// config
	addr            string
	tlsCert         string
	tlsKey          string
	autoTLS         *tls.Certificate
	autocertManager interface{ TLSConfig() *tls.Config } // *autocert.Manager
	certRotator     *CertRotator
	quicConfig      *QUICConfig
	idleTimeout     time.Duration
	checkOrigin     func(r *http.Request) bool

	onConnect     func(*Context)
	onDisconnect  func(*Context)
	shutdownHooks []ShutdownHook
}

// Option configures the Server.
type Option func(*Server)

// New creates a new WebTransport server with the given options.
func New(opts ...Option) *Server {
	s := &Server{
		router:      NewRouter(),
		sessions:    NewSessionStore(),
		addr:        ":4433",
		idleTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithAddr sets the listen address (default ":4433").
func WithAddr(addr string) Option {
	return func(s *Server) { s.addr = addr }
}

// WithTLS sets TLS certificate and key files.
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) {
		s.tlsCert = certFile
		s.tlsKey = keyFile
	}
}

// WithSelfSignedTLS generates a self-signed certificate for development.
// Returns the certificate hash that browsers need for serverCertificateHashes.
func WithSelfSignedTLS() Option {
	return func(s *Server) {
		cert, err := generateSelfSigned()
		if err != nil {
			panic(fmt.Sprintf("wt: failed to generate self-signed cert: %v", err))
		}
		s.autoTLS = cert
	}
}

// WithIdleTimeout sets the session idle timeout (default 30s).
func WithIdleTimeout(d time.Duration) Option {
	return func(s *Server) { s.idleTimeout = d }
}

// WithCheckOrigin sets a function to validate the request origin.
func WithCheckOrigin(fn func(r *http.Request) bool) Option {
	return func(s *Server) { s.checkOrigin = fn }
}

// Use adds global middleware that runs on every session.
func (s *Server) Use(mw ...MiddlewareFunc) {
	s.mw = append(s.mw, mw...)
}

// Handle registers a session handler for the given path pattern.
// The handler receives a Context with the full session — accept streams,
// receive datagrams, access path params, etc.
func (s *Server) Handle(pattern string, handler HandlerFunc, mw ...MiddlewareFunc) {
	s.router.Add(pattern, handler, mw...)
}

// OnConnect registers a callback for new sessions (after middleware).
func (s *Server) OnConnect(fn func(*Context)) {
	s.onConnect = fn
}

// OnDisconnect registers a callback for closed sessions.
func (s *Server) OnDisconnect(fn func(*Context)) {
	s.onDisconnect = fn
}

// Sessions returns the session store for looking up active sessions.
func (s *Server) Sessions() *SessionStore {
	return s.sessions
}

// Broadcast sends a datagram to all active sessions.
func (s *Server) Broadcast(data []byte) {
	s.sessions.Broadcast(data)
}

// BroadcastExcept sends a datagram to all active sessions except the specified one.
func (s *Server) BroadcastExcept(data []byte, excludeID string) {
	s.sessions.Each(func(c *Context) {
		if c.ID() != excludeID {
			_ = c.SendDatagram(data)
		}
	})
}

// SessionCount returns the number of active sessions.
func (s *Server) SessionCount() int {
	return s.sessions.Count()
}

// ListenAndServe starts the WebTransport server.
func (s *Server) ListenAndServe() error {
	tlsCfg, err := s.tlsConfig()
	if err != nil {
		return fmt.Errorf("wt: tls config: %w", err)
	}

	mux := http.NewServeMux()

	h3srv := &http3.Server{
		Addr:      s.addr,
		TLSConfig: tlsCfg,
		Handler:   mux,
	}

	s.wt = &webtransport.Server{
		H3:          h3srv,
		CheckOrigin: s.checkOrigin,
	}

	// Enable HTTP/3 datagram support required by WebTransport
	webtransport.ConfigureHTTP3Server(h3srv)

	// Register all routes as HTTP handlers that upgrade to WebTransport
	for _, route := range s.router.Routes() {
		pattern := route.Pattern
		handler := route.Handler
		routeMW := route.Middleware

		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			session, err := s.wt.Upgrade(w, r)
			if err != nil {
				http.Error(w, "WebTransport upgrade failed", http.StatusInternalServerError)
				return
			}

			params := s.router.ExtractParams(pattern, r.URL.Path)
			ctx := newContext(session, r, params, s)
			s.sessions.Add(ctx)

			defer func() {
				s.sessions.Remove(ctx.ID())
				if s.onDisconnect != nil {
					s.onDisconnect(ctx)
				}
			}()

			if s.onConnect != nil {
				s.onConnect(ctx)
			}

			// Build middleware chain: global + route-specific
			chain := buildChain(handler, append(s.mw, routeMW...))
			chain(ctx)
		})
	}

	// If we have in-memory TLS (self-signed), we need to listen manually
	// since ListenAndServeTLS expects file paths
	if s.autoTLS != nil {
		conn, err := net.ListenPacket("udp", s.addr)
		if err != nil {
			return fmt.Errorf("wt: listen: %w", err)
		}
		return s.wt.Serve(conn)
	}

	return s.wt.ListenAndServeTLS(s.tlsCert, s.tlsKey)
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	if s.wt != nil {
		return s.wt.Close()
	}
	return nil
}

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) tlsConfig() (*tls.Config, error) {
	if s.certRotator != nil {
		return s.certRotator.TLSConfig(), nil
	}
	if s.autocertManager != nil {
		cfg := s.autocertManager.TLSConfig()
		cfg.NextProtos = append(cfg.NextProtos, "h3")
		return cfg, nil
	}
	if s.autoTLS != nil {
		return &tls.Config{
			Certificates: []tls.Certificate{*s.autoTLS},
			NextProtos:   []string{"h3"},
		}, nil
	}
	if s.tlsCert != "" && s.tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(s.tlsCert, s.tlsKey)
		if err != nil {
			return nil, err
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h3"},
		}, nil
	}
	return nil, fmt.Errorf("no TLS configuration: use WithTLS() or WithSelfSignedTLS()")
}

func generateSelfSigned() (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// CertHash returns the SHA-256 hash of the self-signed certificate,
// needed for the browser's serverCertificateHashes option.
// Returns empty string if not using self-signed TLS.
func (s *Server) CertHash() string {
	if s.autoTLS == nil || len(s.autoTLS.Certificate) == 0 {
		return ""
	}
	return CertFingerprint(s.autoTLS.Certificate[0])
}

// Group creates a route group with shared middleware.
func (s *Server) Group(prefix string, mw ...MiddlewareFunc) *Group {
	return &Group{
		server: s,
		prefix: prefix,
		mw:     mw,
	}
}

// Group is a collection of routes that share a path prefix and middleware.
type Group struct {
	server *Server
	prefix string
	mw     []MiddlewareFunc
}

// Handle registers a handler in the group.
func (g *Group) Handle(pattern string, handler HandlerFunc, mw ...MiddlewareFunc) {
	fullPattern := g.prefix + pattern
	allMW := make([]MiddlewareFunc, 0, len(g.mw)+len(mw))
	allMW = append(allMW, g.mw...)
	allMW = append(allMW, mw...)
	g.server.Handle(fullPattern, handler, allMW...)
}

// Use adds middleware to the group.
func (g *Group) Use(mw ...MiddlewareFunc) {
	g.mw = append(g.mw, mw...)
}

// Shutdown gracefully shuts down the server.
// It stops accepting new connections, then waits for active sessions
// to finish or for the context to be cancelled, whichever comes first.
func (s *Server) Shutdown(ctx context.Context) error {
	// Run shutdown hooks
	s.mu.RLock()
	hooks := s.shutdownHooks
	s.mu.RUnlock()
	for _, fn := range hooks {
		fn()
	}

	// Wait for sessions to drain or context to cancel
	done := make(chan struct{})
	go func() {
		for {
			if s.sessions.Count() == 0 {
				close(done)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()

	select {
	case <-done:
		// All sessions drained
	case <-ctx.Done():
		// Timeout — force close remaining sessions
		s.sessions.CloseAll()
	}

	return s.Close()
}
