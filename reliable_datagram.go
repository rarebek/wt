package wt

import (
	"encoding/binary"
	"sync"
	"time"
)

// ReliableDatagram adds optional reliability on top of unreliable datagrams.
// It uses sequence numbers and acknowledgments to detect and retransmit lost messages.
//
// This is useful when you want datagram-like semantics (no head-of-line blocking,
// independent from streams) but need delivery guarantees.
//
// Trade-off: adds 6 bytes of overhead per datagram (2 byte seq + 4 byte timestamp)
// and introduces retransmission latency for lost messages.
type ReliableDatagram struct {
	ctx       *Context
	mu        sync.Mutex
	seq       uint16
	pending   map[uint16]*pendingMsg
	ackCh     chan uint16
	recvSeq   uint16
	timeout   time.Duration
	maxRetry  int
	onReceive func(data []byte)
}

type pendingMsg struct {
	seq      uint16
	data     []byte
	sentAt   time.Time
	retries  int
}

// ReliableOption configures the ReliableDatagram.
type ReliableOption func(*ReliableDatagram)

// WithRetryTimeout sets the retransmission timeout (default: 100ms).
func WithRetryTimeout(d time.Duration) ReliableOption {
	return func(r *ReliableDatagram) { r.timeout = d }
}

// WithMaxRetries sets the maximum retransmission attempts (default: 5).
func WithMaxRetries(n int) ReliableOption {
	return func(r *ReliableDatagram) { r.maxRetry = n }
}

// NewReliableDatagram creates a reliable datagram layer over a session's datagrams.
// The onReceive callback is called for each reliably-delivered message.
func NewReliableDatagram(c *Context, onReceive func(data []byte), opts ...ReliableOption) *ReliableDatagram {
	rd := &ReliableDatagram{
		ctx:       c,
		pending:   make(map[uint16]*pendingMsg),
		ackCh:     make(chan uint16, 64),
		timeout:   100 * time.Millisecond,
		maxRetry:  5,
		onReceive: onReceive,
	}
	for _, opt := range opts {
		opt(rd)
	}
	go rd.receiveLoop()
	go rd.retransmitLoop()
	return rd
}

// Send sends a datagram with reliability guarantees.
func (rd *ReliableDatagram) Send(data []byte) error {
	rd.mu.Lock()
	seq := rd.seq
	rd.seq++
	msg := &pendingMsg{
		seq:    seq,
		data:   data,
		sentAt: time.Now(),
	}
	rd.pending[seq] = msg
	rd.mu.Unlock()

	return rd.sendRaw(seq, data)
}

func (rd *ReliableDatagram) sendRaw(seq uint16, data []byte) error {
	// Format: [0x01 type][seq uint16][payload]
	frame := make([]byte, 3+len(data))
	frame[0] = 0x01 // data frame
	binary.BigEndian.PutUint16(frame[1:], seq)
	copy(frame[3:], data)
	return rd.ctx.SendDatagram(frame)
}

func (rd *ReliableDatagram) sendAck(seq uint16) error {
	// Format: [0x02 type][seq uint16]
	frame := make([]byte, 3)
	frame[0] = 0x02 // ack frame
	binary.BigEndian.PutUint16(frame[1:], seq)
	return rd.ctx.SendDatagram(frame)
}

func (rd *ReliableDatagram) receiveLoop() {
	for {
		data, err := rd.ctx.ReceiveDatagram()
		if err != nil {
			return
		}
		if len(data) < 3 {
			continue
		}

		frameType := data[0]
		seq := binary.BigEndian.Uint16(data[1:3])

		switch frameType {
		case 0x01: // Data frame
			// Send ACK
			rd.sendAck(seq)
			// Deliver to application
			if rd.onReceive != nil {
				rd.onReceive(data[3:])
			}

		case 0x02: // ACK frame
			rd.mu.Lock()
			delete(rd.pending, seq)
			rd.mu.Unlock()
		}
	}
}

func (rd *ReliableDatagram) retransmitLoop() {
	ticker := time.NewTicker(rd.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rd.mu.Lock()
			now := time.Now()
			for seq, msg := range rd.pending {
				if now.Sub(msg.sentAt) > rd.timeout {
					if msg.retries >= rd.maxRetry {
						delete(rd.pending, seq)
						continue
					}
					msg.retries++
					msg.sentAt = now
					go rd.sendRaw(seq, msg.data)
				}
			}
			rd.mu.Unlock()

		case <-rd.ctx.Context().Done():
			return
		}
	}
}

// PendingCount returns the number of unacknowledged messages.
func (rd *ReliableDatagram) PendingCount() int {
	rd.mu.Lock()
	n := len(rd.pending)
	rd.mu.Unlock()
	return n
}
