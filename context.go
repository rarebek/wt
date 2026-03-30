package wt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/webtransport-go"
)

// Context wraps a WebTransport session with routing info, metadata, and helpers.
type Context struct {
	session *webtransport.Session
	request *http.Request
	server  *Server
	params  map[string]string
	mu      sync.RWMutex
	store   map[string]any
	id      string
}

func newContext(session *webtransport.Session, r *http.Request, params map[string]string, s *Server) *Context {
	if params == nil {
		params = make(map[string]string)
	}
	return &Context{
		session: session,
		request: r,
		server:  s,
		params:  params,
		store:   make(map[string]any),
		id:      generateSessionID(session),
	}
}

// Session returns the underlying webtransport.Session.
func (c *Context) Session() *webtransport.Session {
	return c.session
}

// Request returns the original HTTP request that initiated the WebTransport session.
func (c *Context) Request() *http.Request {
	return c.request
}

// Param returns a path parameter value by name.
// For pattern "/chat/{room}" and path "/chat/general", Param("room") returns "general".
func (c *Context) Param(name string) string {
	return c.params[name]
}

// Params returns all path parameters.
func (c *Context) Params() map[string]string {
	return c.params
}

// ID returns a unique identifier for this session.
func (c *Context) ID() string {
	return c.id
}

// Set stores a key-value pair in the context (thread-safe).
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	c.store[key] = value
	c.mu.Unlock()
}

// Get retrieves a value from the context store.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.store[key]
	c.mu.RUnlock()
	return v, ok
}

// MustGet retrieves a value or panics if not found.
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic(fmt.Sprintf("wt: key %q not found in context", key))
	}
	return v
}

// GetString retrieves a string value from the context store.
func (c *Context) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Context returns the session's context (for cancellation/deadline).
func (c *Context) Context() context.Context {
	return c.session.Context()
}

// AcceptStream accepts the next incoming bidirectional stream.
func (c *Context) AcceptStream() (*Stream, error) {
	s, err := c.session.AcceptStream(c.Context())
	if err != nil {
		return nil, err
	}
	return newStream(s, c), nil
}

// OpenStream opens a new bidirectional stream to the client.
func (c *Context) OpenStream() (*Stream, error) {
	s, err := c.session.OpenStream()
	if err != nil {
		return nil, err
	}
	return newStream(s, c), nil
}

// OpenStreamSync opens a new bidirectional stream, blocking until flow control allows.
func (c *Context) OpenStreamSync() (*Stream, error) {
	s, err := c.session.OpenStreamSync(c.Context())
	if err != nil {
		return nil, err
	}
	return newStream(s, c), nil
}

// AcceptUniStream accepts the next incoming unidirectional stream (receive only).
func (c *Context) AcceptUniStream() (*ReceiveStream, error) {
	s, err := c.session.AcceptUniStream(c.Context())
	if err != nil {
		return nil, err
	}
	return &ReceiveStream{raw: s}, nil
}

// OpenUniStream opens a unidirectional stream to the client (send only).
func (c *Context) OpenUniStream() (*SendStream, error) {
	s, err := c.session.OpenUniStream()
	if err != nil {
		return nil, err
	}
	return &SendStream{raw: s}, nil
}

// SendDatagram sends an unreliable datagram to the client.
func (c *Context) SendDatagram(data []byte) error {
	return c.session.SendDatagram(data)
}

// ReceiveDatagram receives the next datagram from the client.
func (c *Context) ReceiveDatagram() ([]byte, error) {
	return c.session.ReceiveDatagram(c.Context())
}

// ReceiveDatagramContext receives a datagram with explicit context for cancellation/timeout.
func (c *Context) ReceiveDatagramContext(ctx context.Context) ([]byte, error) {
	return c.session.ReceiveDatagram(ctx)
}

// Close closes the session with a success code.
func (c *Context) Close() error {
	return c.session.CloseWithError(0, "")
}

// CloseWithError closes the session with an error code and message.
func (c *Context) CloseWithError(code uint32, msg string) error {
	return c.session.CloseWithError(webtransport.SessionErrorCode(code), msg)
}

// LocalAddr returns the server's local address.
func (c *Context) LocalAddr() net.Addr {
	return c.session.LocalAddr()
}

// RemoteAddr returns the client's address.
func (c *Context) RemoteAddr() net.Addr {
	return c.session.RemoteAddr()
}

// Server returns the Server instance.
func (c *Context) Server() *Server {
	return c.server
}

func generateSessionID(s *webtransport.Session) string {
	data := fmt.Sprintf("%s-%s-%p", s.LocalAddr(), s.RemoteAddr(), s)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// CertFingerprint returns the hex-encoded SHA-256 fingerprint of a DER-encoded certificate.
func CertFingerprint(certDER []byte) string {
	hash := sha256.Sum256(certDER)
	return hex.EncodeToString(hash[:])
}

// ConnInfo provides detailed connection information for a session.
type ConnInfo struct {
	SessionID   string            `json:"session_id"`
	RemoteAddr  string            `json:"remote_addr"`
	LocalAddr   string            `json:"local_addr"`
	Path        string            `json:"path"`
	Params      map[string]string `json:"params,omitempty"`
	ConnectedAt time.Time         `json:"connected_at"`
	Transport   string            `json:"transport"` // "webtransport" or "websocket"
	UserAgent   string            `json:"user_agent,omitempty"`
	Origin      string            `json:"origin,omitempty"`
}

// Info returns connection information for the session.
func (c *Context) Info() ConnInfo {
	info := ConnInfo{
		SessionID:  c.ID(),
		RemoteAddr: c.RemoteAddr().String(),
		LocalAddr:  c.LocalAddr().String(),
		Path:       c.Request().URL.Path,
		Params:     c.Params(),
		Transport:  "webtransport",
		UserAgent:  c.Request().UserAgent(),
		Origin:     c.Request().Header.Get("Origin"),
	}

	if t, ok := c.Get("_connected_at"); ok {
		info.ConnectedAt = t.(time.Time)
	}

	return info
}

// InfoJSON returns connection info as a JSON string.
func (c *Context) InfoJSON() string {
	data, _ := json.Marshal(c.Info())
	return string(data)
}

// SendBatch sends multiple datagrams in rapid succession.
// More efficient than calling SendDatagram in a loop because
// it reduces lock contention on the underlying QUIC connection.
func (c *Context) SendBatch(messages [][]byte) int {
	sent := 0
	for _, msg := range messages {
		if err := c.SendDatagram(msg); err != nil {
			break
		}
		sent++
	}
	return sent
}

// SendBatchRoom sends multiple datagrams to all room members.
func (r *Room) SendBatch(messages [][]byte) {
	for _, msg := range messages {
		r.Broadcast(msg)
	}
}

// Priority represents a stream urgency level.
// Higher priority streams are serviced first when resources are constrained.
// Based on HTTP/3 priority signaling (RFC 9218).
type Priority int

const (
	// PriorityBackground is for non-urgent data (analytics, telemetry).
	PriorityBackground Priority = 0
	// PriorityLow is for bulk transfers (file uploads, log streaming).
	PriorityLow Priority = 1
	// PriorityNormal is the default priority.
	PriorityNormal Priority = 3
	// PriorityHigh is for interactive data (chat messages, game events).
	PriorityHigh Priority = 5
	// PriorityCritical is for control messages (auth, heartbeat, disconnect).
	PriorityCritical Priority = 7
)

// StreamConfig holds configuration for a new stream.
type StreamConfig struct {
	// Priority hint for this stream (not enforced by QUIC, but useful for
	// application-level scheduling).
	Priority Priority

	// TypeID is the StreamMux type identifier (0 = no mux).
	TypeID uint16
}

// DefaultStreamConfig returns default stream configuration.
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		Priority: PriorityNormal,
		TypeID:   0,
	}
}

// Validator can validate itself. Implement on message types for automatic validation.
type Validator interface {
	Validate() error
}

// ValidateMessage checks if a decoded message implements Validator and validates it.
// Returns nil if the message doesn't implement Validator.
func ValidateMessage(msg any) error {
	if v, ok := msg.(Validator); ok {
		return v.Validate()
	}
	return nil
}

// RequiredFields checks that the given struct fields are non-zero.
// Useful for simple message validation without a full validation library.
//
// Usage:
//
//	type ChatMsg struct {
//	    User string
//	    Text string
//	}
//	func (m ChatMsg) Validate() error {
//	    return wt.RequiredFields(m, "User", "Text")
//	}
func RequiredFields(v any, fields ...string) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("wt: RequiredFields expects a struct, got %T", v)
	}

	var missing []string
	for _, name := range fields {
		f := val.FieldByName(name)
		if !f.IsValid() {
			missing = append(missing, name)
			continue
		}
		if f.IsZero() {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("wt: required fields missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ServerInfo returns information about the framework and runtime.
func ServerInfo() map[string]string {
	return map[string]string{
		"framework":  "wt",
		"version":    "0.1.0-dev",
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       fmt.Sprintf("%d", runtime.NumCPU()),
	}
}

// Hash returns the SHA-256 hash of the given data as a hex string.
func Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// JoinPath joins path segments with / separators, cleaning doubles.
func JoinPath(segments ...string) string {
	joined := strings.Join(segments, "/")
	// Clean double slashes
	for strings.Contains(joined, "//") {
		joined = strings.ReplaceAll(joined, "//", "/")
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return strings.TrimSuffix(joined, "/")
}

// Must panics if err is non-nil. Useful for initialization.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("wt: %v", err))
	}
	return val
}
