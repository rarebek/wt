// Package client provides a WebTransport client with reconnection support.
package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/webtransport-go"
	"github.com/rarebek/wt/codec"
)

// Client is a WebTransport client with auto-reconnection.
type Client struct {
	dialer  *webtransport.Dialer
	session *webtransport.Session
	mu      sync.RWMutex
	url     string
	headers http.Header
	codec   codec.Codec

	// Reconnection
	reconnect       bool
	reconnectMin    time.Duration
	reconnectMax    time.Duration
	onReconnect     func()
	onDisconnect    func(error)
}

// Option configures the Client.
type Option func(*Client)

// WithReconnect enables auto-reconnection with exponential backoff.
func WithReconnect(min, max time.Duration) Option {
	return func(c *Client) {
		c.reconnect = true
		c.reconnectMin = min
		c.reconnectMax = max
	}
}

// WithHeaders sets custom headers for the WebTransport handshake.
func WithHeaders(h http.Header) Option {
	return func(c *Client) { c.headers = h }
}

// WithCodec sets the message codec (default: JSON).
func WithCodec(cc codec.Codec) Option {
	return func(c *Client) { c.codec = cc }
}

// WithInsecureSkipVerify disables TLS certificate verification (for development).
func WithInsecureSkipVerify() Option {
	return func(c *Client) {
		c.dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

// OnReconnect sets a callback for successful reconnections.
func OnReconnect(fn func()) Option {
	return func(c *Client) { c.onReconnect = fn }
}

// OnDisconnect sets a callback for disconnections.
func OnDisconnect(fn func(error)) Option {
	return func(c *Client) { c.onDisconnect = fn }
}

// New creates a new WebTransport client.
func New(url string, opts ...Option) *Client {
	c := &Client{
		dialer:       &webtransport.Dialer{},
		url:          url,
		headers:      http.Header{},
		codec:        codec.JSON{},
		reconnectMin: 1 * time.Second,
		reconnectMax: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Dial connects to the WebTransport server.
func (c *Client) Dial(ctx context.Context) error {
	_, session, err := c.dialer.Dial(ctx, c.url, c.headers)
	if err != nil {
		return fmt.Errorf("wt client: dial: %w", err)
	}
	c.mu.Lock()
	c.session = session
	c.mu.Unlock()
	return nil
}

// Session returns the underlying session.
func (c *Client) Session() *webtransport.Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session
}

// OpenStream opens a bidirectional stream.
func (c *Client) OpenStream(ctx context.Context) (*webtransport.Stream, error) {
	s := c.Session()
	if s == nil {
		return nil, fmt.Errorf("wt client: not connected")
	}
	return s.OpenStreamSync(ctx)
}

// AcceptStream accepts an incoming stream from the server.
func (c *Client) AcceptStream(ctx context.Context) (*webtransport.Stream, error) {
	s := c.Session()
	if s == nil {
		return nil, fmt.Errorf("wt client: not connected")
	}
	return s.AcceptStream(ctx)
}

// SendDatagram sends an unreliable datagram.
func (c *Client) SendDatagram(data []byte) error {
	s := c.Session()
	if s == nil {
		return fmt.Errorf("wt client: not connected")
	}
	return s.SendDatagram(data)
}

// ReceiveDatagram receives a datagram.
func (c *Client) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	s := c.Session()
	if s == nil {
		return nil, fmt.Errorf("wt client: not connected")
	}
	return s.ReceiveDatagram(ctx)
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil {
		err := c.session.CloseWithError(0, "")
		c.session = nil
		return err
	}
	return nil
}

// DialWithReconnect connects and automatically reconnects on disconnection.
// Blocks until the context is cancelled.
func (c *Client) DialWithReconnect(ctx context.Context) error {
	backoff := c.reconnectMin

	for {
		err := c.Dial(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			time.Sleep(backoff)
			backoff = min(backoff*2, c.reconnectMax)
			continue
		}

		// Reset backoff on successful connection
		backoff = c.reconnectMin
		if c.onReconnect != nil {
			c.onReconnect()
		}

		// Wait for session to close
		<-c.Session().Context().Done()

		if c.onDisconnect != nil {
			c.onDisconnect(c.Session().Context().Err())
		}

		if !c.reconnect || ctx.Err() != nil {
			return ctx.Err()
		}

		time.Sleep(backoff)
		backoff = min(backoff*2, c.reconnectMax)
	}
}
