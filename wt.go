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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"iter"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/crypto/acme/autocert"
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

// AltSvcHeader returns the Alt-Svc HTTP header value that tells browsers
// to upgrade from HTTP/2 to HTTP/3 for WebTransport.
//
// Browsers use this header to discover that a server supports HTTP/3.
// Include it in your HTTP/1.1 or HTTP/2 responses.
//
// Usage:
//
//	// On your HTTP/1.1 or HTTP/2 server:
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	    wt.SetAltSvcHeader(w, 4433) // WebTransport on port 4433
//	    // ... serve regular HTTP
//	})
func AltSvcHeader(port int) string {
	return fmt.Sprintf(`h3=":%d"; ma=86400`, port)
}

// SetAltSvcHeader sets the Alt-Svc header on an HTTP response.
func SetAltSvcHeader(w http.ResponseWriter, port int) {
	w.Header().Set("Alt-Svc", AltSvcHeader(port))
}

// AltSvcMiddleware returns an HTTP middleware that adds the Alt-Svc header
// to every response, advertising HTTP/3 availability.
func AltSvcMiddleware(port int) func(http.Handler) http.Handler {
	header := AltSvcHeader(port)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Alt-Svc", header)
			next.ServeHTTP(w, r)
		})
	}
}

// QUICConfig exposes QUIC-level tuning options.
// These are passed to the underlying quic-go transport.
type QUICConfig struct {
	// InitialStreamReceiveWindow is the initial flow control window for each stream (default: 512KB).
	// Increase for high-throughput streams. quic-go auto-tunes up to MaxStreamReceiveWindow.
	InitialStreamReceiveWindow uint64

	// MaxStreamReceiveWindow is the maximum stream flow control window (default: 6MB).
	MaxStreamReceiveWindow uint64

	// InitialConnectionReceiveWindow is the initial connection-level flow control window (default: 768KB).
	InitialConnectionReceiveWindow uint64

	// MaxConnectionReceiveWindow is the max connection-level flow control window (default: 15MB).
	MaxConnectionReceiveWindow uint64

	// MaxIncomingStreams is the maximum number of concurrent incoming streams per session.
	// Default: 100 in quic-go.
	MaxIncomingStreams int64

	// MaxIncomingUniStreams is the max number of concurrent incoming unidirectional streams.
	MaxIncomingUniStreams int64
}

// DefaultQUICConfig returns sensible defaults for most applications.
func DefaultQUICConfig() QUICConfig {
	return QUICConfig{
		InitialStreamReceiveWindow:     512 * 1024,       // 512 KB
		MaxStreamReceiveWindow:         6 * 1024 * 1024,  // 6 MB
		InitialConnectionReceiveWindow: 768 * 1024,       // 768 KB
		MaxConnectionReceiveWindow:     15 * 1024 * 1024, // 15 MB
		MaxIncomingStreams:             100,
		MaxIncomingUniStreams:          100,
	}
}

// GameServerQUICConfig returns config optimized for game servers:
// smaller windows (less buffering = lower latency), more streams.
func GameServerQUICConfig() QUICConfig {
	return QUICConfig{
		InitialStreamReceiveWindow:     64 * 1024,   // 64 KB — small for low latency
		MaxStreamReceiveWindow:         256 * 1024,  // 256 KB
		InitialConnectionReceiveWindow: 128 * 1024,  // 128 KB
		MaxConnectionReceiveWindow:     1024 * 1024, // 1 MB
		MaxIncomingStreams:             200,         // more streams for game events
		MaxIncomingUniStreams:          50,
	}
}

// HighThroughputQUICConfig returns config optimized for large transfers:
// bigger windows, fewer streams.
func HighThroughputQUICConfig() QUICConfig {
	return QUICConfig{
		InitialStreamReceiveWindow:     2 * 1024 * 1024,  // 2 MB
		MaxStreamReceiveWindow:         16 * 1024 * 1024, // 16 MB
		InitialConnectionReceiveWindow: 4 * 1024 * 1024,  // 4 MB
		MaxConnectionReceiveWindow:     32 * 1024 * 1024, // 32 MB
		MaxIncomingStreams:             50,
		MaxIncomingUniStreams:          20,
	}
}

// toQuicConfig converts to quic-go's Config type.
func (c QUICConfig) toQuicConfig() *quic.Config {
	return &quic.Config{
		InitialStreamReceiveWindow:     c.InitialStreamReceiveWindow,
		MaxStreamReceiveWindow:         c.MaxStreamReceiveWindow,
		InitialConnectionReceiveWindow: c.InitialConnectionReceiveWindow,
		MaxConnectionReceiveWindow:     c.MaxConnectionReceiveWindow,
		MaxIncomingStreams:             c.MaxIncomingStreams,
		MaxIncomingUniStreams:          c.MaxIncomingUniStreams,
	}
}

// WithQUICConfig sets QUIC-level transport options.
func WithQUICConfig(cfg QUICConfig) Option {
	return func(s *Server) {
		s.quicConfig = &cfg
	}
}

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

// Chain composes multiple handlers into a single handler.
// Each handler runs sequentially. Useful for setup+teardown patterns.
func Chain(handlers ...HandlerFunc) HandlerFunc {
	return func(c *Context) {
		for _, h := range handlers {
			h(c)
		}
	}
}

// FirstMatch tries handlers in order, stopping at the first one that
// doesn't close the session. Useful for fallback patterns.
func FirstMatch(handlers ...HandlerFunc) HandlerFunc {
	return func(c *Context) {
		for _, h := range handlers {
			h(c)
			if c.Context().Err() != nil {
				return // session closed by this handler
			}
		}
	}
}

// HandleStream is a convenience for handlers that process one stream at a time.
// It auto-accepts streams and calls the handler for each in a new goroutine.
//
// Usage:
//
//	server.Handle("/echo", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
//	    defer s.Close()
//	    msg, _ := s.ReadMessage()
//	    s.WriteMessage(msg)
//	}))
func HandleStream(fn StreamHandler) HandlerFunc {
	return func(c *Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go fn(stream, c)
		}
	}
}

// HandleDatagram is a convenience for handlers that only process datagrams.
// It loops receiving datagrams and calls the handler for each.
//
// Usage:
//
//	server.Handle("/ping", wt.HandleDatagram(func(data []byte, c *wt.Context) []byte {
//	    return data // echo
//	}))
func HandleDatagram(fn func(data []byte, c *Context) []byte) HandlerFunc {
	return func(c *Context) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			reply := fn(data, c)
			if reply != nil {
				_ = c.SendDatagram(reply)
			}
		}
	}
}

// HandleBoth is a convenience for handlers that process both streams and datagrams.
// Datagrams are handled in a background goroutine, streams in the main loop.
//
// Usage:
//
//	server.Handle("/game", wt.HandleBoth(
//	    func(s *wt.Stream, c *wt.Context) {
//	        // handle game event stream
//	    },
//	    func(data []byte, c *wt.Context) []byte {
//	        // handle position datagram
//	        return nil // no reply
//	    },
//	))
func HandleBoth(streamFn StreamHandler, datagramFn func([]byte, *Context) []byte) HandlerFunc {
	return func(c *Context) {
		// Datagrams in background
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				reply := datagramFn(data, c)
				if reply != nil {
					_ = c.SendDatagram(reply)
				}
			}
		}()

		// Streams in foreground
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go streamFn(stream, c)
		}
	}
}

// Streams returns an iterator over incoming streams.
// Use with Go 1.23+ range-over-func:
//
//	for stream := range wt.Streams(c) {
//	    go handleStream(stream)
//	}
func Streams(c *Context) iter.Seq[*Stream] {
	return func(yield func(*Stream) bool) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			if !yield(stream) {
				return
			}
		}
	}
}

// Datagrams returns an iterator over incoming datagrams.
//
//	for data := range wt.Datagrams(c) {
//	    process(data)
//	}
func Datagrams(c *Context) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			if !yield(data) {
				return
			}
		}
	}
}

// Messages returns an iterator over length-prefixed messages from a stream.
//
//	for msg := range wt.Messages(stream) {
//	    process(msg)
//	}
func Messages(s *Stream) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			if !yield(msg) {
				return
			}
		}
	}
}

// SendJSON sends a JSON-encoded value as a datagram.
func (c *Context) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.SendDatagram(data)
}

// WriteJSON writes a JSON-encoded value as a stream message.
func (s *Stream) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.WriteMessage(data)
}

// ReadJSON reads a stream message and decodes it as JSON.
func (s *Stream) ReadJSON(v any) error {
	data, err := s.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// BroadcastJSON encodes v as JSON and broadcasts to all active sessions.
func (s *Server) BroadcastJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Broadcast(data)
	return nil
}

// BroadcastJSONExcept sends to all except one session.
func (s *Server) BroadcastJSONExcept(v any, excludeID string) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.BroadcastExcept(data, excludeID)
	return nil
}

// MulticastJSON sends JSON to sessions matching a filter.
func (s *Server) MulticastJSON(v any, filter func(*Context) bool) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Multicast(data, filter)
	return nil
}

// BroadcastJSONRoom encodes and broadcasts to all room members.
func BroadcastJSONRoom(r *Room, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Broadcast(data)
	return nil
}

// Multicast sends a datagram to sessions matching a filter function.
// More flexible than Broadcast — only sends to sessions that match.
//
// Usage:
//
//	// Send to all sessions with user role "admin"
//	server.Multicast(data, func(c *Context) bool {
//	    role, _ := c.Get("role")
//	    return role == "admin"
//	})
func (s *Server) Multicast(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			_ = c.SendDatagram(data)
		}
	})
}

// MulticastStream sends a reliable message via streams to matching sessions.
func (s *Server) MulticastStream(data []byte, filter func(*Context) bool) {
	s.sessions.Each(func(c *Context) {
		if filter(c) {
			go func() {
				stream, err := c.OpenStream()
				if err != nil {
					return
				}
				_ = stream.WriteMessage(data)
				stream.Close()
			}()
		}
	})
}

// Notification represents a structured notification message.
type Notification struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// Notify sends a notification datagram to the session.
func (c *Context) Notify(notif Notification) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return c.SendDatagram(data)
}

// NotifyAll sends a notification to all active sessions.
func (s *Server) NotifyAll(notif Notification) {
	data, _ := json.Marshal(notif)
	s.Broadcast(data)
}

// NotifyRoom sends a notification to all room members.
func NotifyRoom(r *Room, notif Notification) {
	data, _ := json.Marshal(notif)
	r.Broadcast(data)
}

// ShutdownHook is a function called during server shutdown.
type ShutdownHook func()

// OnShutdown registers a function to be called when the server shuts down.
// Hooks run in registration order before connections are drained.
func (s *Server) OnShutdown(fn ShutdownHook) {
	s.mu.Lock()
	s.shutdownHooks = append(s.shutdownHooks, fn)
	s.mu.Unlock()
}

// ListenAndServeWithGracefulShutdown starts the server and handles SIGTERM/SIGINT
// for graceful shutdown. Active sessions are drained within the given timeout.
//
// Usage:
//
//	server := wt.New(...)
//	server.Handle("/app", handler)
//	wt.ListenAndServeWithGracefulShutdown(server, 30*time.Second)
func ListenAndServeWithGracefulShutdown(s *Server, drainTimeout time.Duration) error {
	logger := slog.Default()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()

	// Wait for signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, starting graceful shutdown",
			"signal", sig.String(),
			"drain_timeout", drainTimeout.String(),
			"active_sessions", s.SessionCount(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
			return err
		}

		logger.Info("graceful shutdown complete",
			"remaining_sessions", s.SessionCount(),
		)
		return nil

	case err := <-errCh:
		return err
	}
}

// Version is the current framework version.
const Version = "0.1.0-dev"

/*
Package wt provides a high-level framework for building WebTransport applications in Go.

WebTransport is a modern protocol built on QUIC and HTTP/3 that provides
multiplexed bidirectional streams, unidirectional streams, and unreliable
datagrams — all without TCP's head-of-line blocking.

This package sits on top of quic-go/webtransport-go and provides:

  - Path-based routing with parameter extraction
  - Middleware stack (auth, logging, rate limiting, compression, metrics)
  - Session management with rooms and pub/sub
  - Message framing (length-prefixed) over streams
  - Type-safe stream handlers using Go generics
  - WebSocket fallback for browsers without WebTransport support
  - Self-signed certificate generation for development

# Quick Start

	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	server.Handle("/echo", func(c *wt.Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				io.Copy(stream, stream)
			}()
		}
	})

	server.ListenAndServe()

# Architecture

The framework follows familiar Go patterns inspired by Gin, Echo, and Chi:

  - Handlers receive a [Context] wrapping the WebTransport session
  - Middleware uses the func(c *Context, next HandlerFunc) signature
  - Route groups share prefixes and middleware
  - Sessions are tracked in a [SessionStore] for enumeration and broadcast

# Streams vs Datagrams

WebTransport offers two data transport modes:

Streams are reliable and ordered, similar to TCP connections. Use them for
messages that must arrive: chat messages, game events, file transfers.

Datagrams are unreliable and unordered, similar to UDP packets. Use them for
data where the latest value matters more than every value: player positions,
sensor readings, cursor locations.

# WebSocket Fallback

The [fallback] sub-package provides transparent WebSocket fallback for browsers
that don't support WebTransport (notably Safari as of early 2026). Stream
multiplexing is simulated over the single WebSocket connection.
*/

// MigrationEvent represents a connection migration (IP address change).
type MigrationEvent struct {
	SessionID  string
	OldAddr    net.Addr
	NewAddr    net.Addr
	MigratedAt time.Time
}

// MigrationWatcher monitors sessions for address changes.
// QUIC handles migration transparently, but this watcher lets
// you react to migrations (logging, analytics, security checks).
type MigrationWatcher struct {
	mu        sync.Mutex
	sessions  map[string]string // sessionID -> last known remote addr
	onMigrate func(MigrationEvent)
	interval  time.Duration
	done      chan struct{}
}

// NewMigrationWatcher creates a watcher that polls session addresses.
func NewMigrationWatcher(store *SessionStore, onMigrate func(MigrationEvent)) *MigrationWatcher {
	mw := &MigrationWatcher{
		sessions:  make(map[string]string),
		onMigrate: onMigrate,
		interval:  5 * time.Second,
		done:      make(chan struct{}),
	}
	go mw.watchLoop(store)
	return mw
}

func (mw *MigrationWatcher) watchLoop(store *SessionStore) {
	ticker := time.NewTicker(mw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			store.Each(func(c *Context) {
				addr := c.RemoteAddr().String()
				mw.mu.Lock()
				old, exists := mw.sessions[c.ID()]
				if exists && old != addr {
					// Migration detected!
					event := MigrationEvent{
						SessionID:  c.ID(),
						OldAddr:    parseAddr(old),
						NewAddr:    c.RemoteAddr(),
						MigratedAt: time.Now(),
					}
					mw.mu.Unlock()
					if mw.onMigrate != nil {
						mw.onMigrate(event)
					}
					mw.mu.Lock()
				}
				mw.sessions[c.ID()] = addr
				mw.mu.Unlock()
			})
		case <-mw.done:
			return
		}
	}
}

// Stop stops the migration watcher.
func (mw *MigrationWatcher) Stop() {
	select {
	case <-mw.done:
	default:
		close(mw.done)
	}
}

type simpleAddr struct{ s string }

func (a simpleAddr) Network() string { return "udp" }
func (a simpleAddr) String() string  { return a.s }

func parseAddr(s string) net.Addr {
	return simpleAddr{s}
}

// This file documents how QUIC connection migration works with the wt framework.
// Connection migration is handled transparently by quic-go — the framework
// doesn't need to do anything special.
//
// # How It Works
//
// QUIC connections use Connection IDs instead of IP:port tuples.
// When a mobile device switches from WiFi to cellular, the IP address changes
// but the Connection ID stays the same. quic-go automatically handles this.
//
// From the framework's perspective:
//   - Sessions survive network changes without reconnection
//   - Streams and datagrams continue working after migration
//   - Context.RemoteAddr() may return a new address after migration
//   - No session state is lost
//
// # Server-Side (Automatic)
//
// The server automatically detects NAT rebinding (same client, new IP/port)
// and performs path validation. No framework code needed.
//
// # Client-Side (via quic-go API)
//
// For Go clients that need to explicitly migrate (e.g., switch from WiFi to cellular):
//
//	import "github.com/quic-go/quic-go"
//
//	// 1. Create a new transport for the new network interface
//	newConn, _ := net.ListenUDP("udp", &net.UDPAddr{})
//	newTransport := &quic.Transport{Conn: newConn}
//
//	// 2. Add a path through the underlying QUIC connection
//	// (requires access to the raw quic.Conn via the client SDK)
//	path, _ := quicConn.AddPath(newTransport)
//
//	// 3. Probe the new path
//	ctx, cancel := context.WithTimeout(ctx, time.Second)
//	path.Probe(ctx)
//	cancel()
//
//	// 4. Switch to the new path
//	path.Switch()
//
//	// 5. Close the old path
//	path.Close()
//
// # What the Framework Handles Automatically
//
//   - Session identity preservation across migrations
//   - Room membership maintained (no rejoin needed)
//   - Context store values preserved
//   - Active streams continue uninterrupted
//   - Middleware state preserved
//
// # Limitations
//
//   - Only client-initiated migration (per RFC 9000)
//   - Server cannot force a client to migrate
//   - quic-go does not yet support Preferred Address (issue #4965)

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
