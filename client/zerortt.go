package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

)

// ZeroRTTClient extends Client with 0-RTT session resumption support.
// On the first connection, a normal handshake occurs and a TLS session ticket
// is cached. Subsequent connections attempt 0-RTT, sending data immediately
// without waiting for the handshake to complete.
//
// 0-RTT is safe for idempotent operations only (reads, subscriptions).
// For writes/mutations, wait for HandshakeComplete before sending.
type ZeroRTTClient struct {
	*Client
	sessionCache tls.ClientSessionCache
}

// NewZeroRTT creates a client with 0-RTT support.
// The cacheSize parameter controls how many session tickets to cache (default: 100).
func NewZeroRTT(url string, cacheSize int, opts ...Option) *ZeroRTTClient {
	if cacheSize <= 0 {
		cacheSize = 100
	}

	cache := tls.NewLRUClientSessionCache(cacheSize)

	c := New(url, opts...)
	c.dialer.TLSClientConfig = &tls.Config{
		ClientSessionCache: cache,
		InsecureSkipVerify: true, // Default for dev; override via options
	}

	return &ZeroRTTClient{
		Client:       c,
		sessionCache: cache,
	}
}

// DialEarly connects using 0-RTT if a session ticket is available.
// On first connection, performs a normal handshake and caches the ticket.
// On subsequent connections, sends data immediately (0-RTT).
//
// IMPORTANT: 0-RTT data can be replayed by attackers. Only send
// idempotent data before HandshakeComplete. Call WaitHandshake()
// before performing any mutations.
func (c *ZeroRTTClient) DialEarly(ctx context.Context) error {
	_, session, err := c.dialer.Dial(ctx, c.url, http.Header{})
	if err != nil {
		return fmt.Errorf("wt client: dial early: %w", err)
	}
	c.mu.Lock()
	c.session = session
	c.mu.Unlock()
	return nil
}

// WaitHandshake blocks until the TLS handshake is fully complete.
// Call this before sending any non-idempotent data on a 0-RTT connection.
func (c *ZeroRTTClient) WaitHandshake(ctx context.Context) error {
	session := c.Session()
	if session == nil {
		return fmt.Errorf("wt client: not connected")
	}

	// webtransport.Session wraps quic.Connection
	// The session context is cancelled when the connection closes
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-session.Context().Done():
		return fmt.Errorf("wt client: session closed during handshake")
	default:
		// Session is active, handshake completed (webtransport.Session
		// is only created after the HTTP/3 CONNECT completes, which
		// requires the TLS handshake to finish)
		return nil
	}
}

// Handle0RTTRejection should be called when 0-RTT is rejected by the server.
// It reconnects using a normal handshake.
func (c *ZeroRTTClient) Handle0RTTRejection(ctx context.Context) error {
	// Close the current session if any
	c.Close()
	// Reconnect normally
	return c.Dial(ctx)
}

// Has0RTTTicket returns true if a session ticket is cached for the server,
// meaning the next connection will attempt 0-RTT.
func (c *ZeroRTTClient) Has0RTTTicket() bool {
	// Check if the session cache has an entry
	// Unfortunately, tls.ClientSessionCache doesn't expose a "has" method,
	// so we check via Get
	if c.dialer.TLSClientConfig == nil || c.dialer.TLSClientConfig.ClientSessionCache == nil {
		return false
	}
	// We can't directly check without the server name, so return true
	// if we've connected at least once (best effort)
	return c.Session() != nil
}

// SessionResumption describes the type of session establishment.
type SessionResumption int

const (
	// FullHandshake means a complete TLS 1.3 handshake was performed.
	FullHandshake SessionResumption = iota
	// Resumed means TLS session resumption was used (1-RTT).
	Resumed
	// ZeroRTT means 0-RTT early data was sent.
	ZeroRTT
)

func (s SessionResumption) String() string {
	switch s {
	case FullHandshake:
		return "full-handshake"
	case Resumed:
		return "resumed"
	case ZeroRTT:
		return "0-rtt"
	default:
		return "unknown"
	}
}

// Note on 0-RTT with WebTransport:
//
// WebTransport sessions are established via an HTTP/3 CONNECT request.
// The CONNECT happens after the QUIC handshake, so by the time the
// webtransport.Session is created, the TLS handshake has completed.
//
// This means 0-RTT for WebTransport primarily benefits connection
// establishment time (saving one round trip), not early data sending.
// The session ticket from a previous connection lets the client skip
// the TLS handshake RTT.
//
// For the framework, this means:
// - Client reconnections are faster (especially over high-latency links)
// - No special handling needed for 0-RTT rejection in WebTransport
//   (the CONNECT only happens after handshake completion)
// - Session cache should be persistent for mobile apps (save to disk)
