package wt

import "fmt"

// Standard error codes for WebTransport sessions and streams.
// HTTP-like codes (0-999) for familiar semantics.
// Framework codes (0x1000+) for wt-specific errors.
// QUIC transport codes (0x100+) are defined by RFC 9000 Section 20.
const (
	CodeOK               uint32 = 0     // Success / normal closure
	CodeUnauthorized     uint32 = 401   // Missing or invalid authentication
	CodeForbidden        uint32 = 403   // Authenticated but not allowed
	CodeNotFound         uint32 = 404   // Route not found
	CodeTimeout          uint32 = 408   // Session or stream timeout
	CodeTooManyRequests  uint32 = 429   // Rate limit exceeded
	CodeInternalError    uint32 = 500   // Server-side error
	CodeServiceUnavail   uint32 = 503   // Server at capacity / circuit open

	// Framework error codes (0x1000+)
	CodeProtocolError    uint32 = 0x1000 // Invalid protocol message
	CodeMessageTooLarge  uint32 = 0x1001 // Message exceeds MaxMessageSize
	CodeInvalidCodec     uint32 = 0x1002 // Unknown or misconfigured codec
	CodeStreamLimit      uint32 = 0x1003 // Too many concurrent streams
	CodeSessionExpired   uint32 = 0x1004 // Session TTL expired
	CodeBadRequest       uint32 = 0x1005 // Malformed request data
	CodeShuttingDown     uint32 = 0x1006 // Server is draining connections
)

// SessionCloseError represents a session closure with a code and message.
type SessionCloseError struct {
	Code    uint32
	Message string
}

func (e *SessionCloseError) Error() string {
	return fmt.Sprintf("wt: session closed with code %d: %s", e.Code, e.Message)
}

// StreamCloseError represents a stream closure with an error code.
type StreamCloseError struct {
	Code   uint32
	Remote bool // true if the remote side closed
}

func (e *StreamCloseError) Error() string {
	side := "local"
	if e.Remote {
		side = "remote"
	}
	return fmt.Sprintf("wt: stream closed by %s with code %d", side, e.Code)
}

// IsSessionClosed checks if an error is a session closure.
func IsSessionClosed(err error) bool {
	_, ok := err.(*SessionCloseError)
	return ok
}

// IsStreamClosed checks if an error is a stream closure.
func IsStreamClosed(err error) bool {
	_, ok := err.(*StreamCloseError)
	return ok
}

// ConnectionError represents a connection-level failure.
type ConnectionError struct {
	Op      string // "dial", "accept", "handshake"
	Addr    string
	Wrapped error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("wt: %s %s: %v", e.Op, e.Addr, e.Wrapped)
}

func (e *ConnectionError) Unwrap() error {
	return e.Wrapped
}

// UpgradeError occurs when WebTransport upgrade fails.
type UpgradeError struct {
	StatusCode int
	Message    string
}

func (e *UpgradeError) Error() string {
	return fmt.Sprintf("wt: upgrade failed with status %d: %s", e.StatusCode, e.Message)
}

// MessageError occurs during message read/write operations.
type MessageError struct {
	Op      string // "read", "write"
	Size    int
	Wrapped error
}

func (e *MessageError) Error() string {
	return fmt.Sprintf("wt: %s message (size=%d): %v", e.Op, e.Size, e.Wrapped)
}

func (e *MessageError) Unwrap() error {
	return e.Wrapped
}

// IsConnectionError checks if an error is a connection-level failure.
func IsConnectionError(err error) bool {
	_, ok := err.(*ConnectionError)
	return ok
}

// IsUpgradeError checks if an error is an upgrade failure.
func IsUpgradeError(err error) bool {
	_, ok := err.(*UpgradeError)
	return ok
}

// IsMessageError checks if an error is a message read/write failure.
func IsMessageError(err error) bool {
	_, ok := err.(*MessageError)
	return ok
}
