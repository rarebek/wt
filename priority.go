package wt

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
