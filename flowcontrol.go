package wt

import (
	"sync/atomic"
)

// FlowControlMonitor tracks stream and datagram flow control metrics.
// Useful for monitoring backpressure and identifying slow consumers.
type FlowControlMonitor struct {
	StreamsOpened   atomic.Int64
	StreamsClosed   atomic.Int64
	DatagramsSent   atomic.Int64
	DatagramsRecvd  atomic.Int64
	BytesSent       atomic.Int64
	BytesReceived   atomic.Int64
	WriteBlocks     atomic.Int64 // times a write was blocked by flow control
}

// NewFlowControlMonitor creates a new monitor.
func NewFlowControlMonitor() *FlowControlMonitor {
	return &FlowControlMonitor{}
}

// FlowStats returns a snapshot of flow control metrics.
type FlowStats struct {
	StreamsOpened  int64 `json:"streams_opened"`
	StreamsClosed  int64 `json:"streams_closed"`
	StreamsActive  int64 `json:"streams_active"`
	DatagramsSent  int64 `json:"datagrams_sent"`
	DatagramsRecvd int64 `json:"datagrams_received"`
	BytesSent      int64 `json:"bytes_sent"`
	BytesReceived  int64 `json:"bytes_received"`
	WriteBlocks    int64 `json:"write_blocks"`
}

// Stats returns current metrics.
func (fc *FlowControlMonitor) Stats() FlowStats {
	opened := fc.StreamsOpened.Load()
	closed := fc.StreamsClosed.Load()
	return FlowStats{
		StreamsOpened:  opened,
		StreamsClosed:  closed,
		StreamsActive:  opened - closed,
		DatagramsSent:  fc.DatagramsSent.Load(),
		DatagramsRecvd: fc.DatagramsRecvd.Load(),
		BytesSent:      fc.BytesSent.Load(),
		BytesReceived:  fc.BytesReceived.Load(),
		WriteBlocks:    fc.WriteBlocks.Load(),
	}
}
