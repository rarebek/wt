package wt

import (
	"github.com/quic-go/quic-go"
)

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
		InitialStreamReceiveWindow:     512 * 1024,  // 512 KB
		MaxStreamReceiveWindow:         6 * 1024 * 1024,   // 6 MB
		InitialConnectionReceiveWindow: 768 * 1024,  // 768 KB
		MaxConnectionReceiveWindow:     15 * 1024 * 1024,  // 15 MB
		MaxIncomingStreams:              100,
		MaxIncomingUniStreams:           100,
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
		MaxIncomingStreams:              200,         // more streams for game events
		MaxIncomingUniStreams:           50,
	}
}

// HighThroughputQUICConfig returns config optimized for large transfers:
// bigger windows, fewer streams.
func HighThroughputQUICConfig() QUICConfig {
	return QUICConfig{
		InitialStreamReceiveWindow:     2 * 1024 * 1024,  // 2 MB
		MaxStreamReceiveWindow:         16 * 1024 * 1024,  // 16 MB
		InitialConnectionReceiveWindow: 4 * 1024 * 1024,   // 4 MB
		MaxConnectionReceiveWindow:     32 * 1024 * 1024,  // 32 MB
		MaxIncomingStreams:              50,
		MaxIncomingUniStreams:           20,
	}
}

// toQuicConfig converts to quic-go's Config type.
func (c QUICConfig) toQuicConfig() *quic.Config {
	return &quic.Config{
		InitialStreamReceiveWindow:     c.InitialStreamReceiveWindow,
		MaxStreamReceiveWindow:         c.MaxStreamReceiveWindow,
		InitialConnectionReceiveWindow: c.InitialConnectionReceiveWindow,
		MaxConnectionReceiveWindow:     c.MaxConnectionReceiveWindow,
		MaxIncomingStreams:              c.MaxIncomingStreams,
		MaxIncomingUniStreams:           c.MaxIncomingUniStreams,
	}
}

// WithQUICConfig sets QUIC-level transport options.
func WithQUICConfig(cfg QUICConfig) Option {
	return func(s *Server) {
		s.quicConfig = &cfg
	}
}
