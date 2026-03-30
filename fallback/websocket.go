// Package fallback provides a transparent WebSocket fallback for the wt framework.
//
// When a client can't use WebTransport (Safari, corporate firewalls blocking UDP,
// old browsers), the fallback layer serves the same handlers over WebSocket.
// Stream multiplexing is simulated over the single WebSocket connection using
// a simple framing protocol.
package fallback

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

// Transport identifies the connection type.
type Transport int

const (
	TransportWebTransport Transport = iota
	TransportWebSocket
)

func (t Transport) String() string {
	switch t {
	case TransportWebTransport:
		return "webtransport"
	case TransportWebSocket:
		return "websocket"
	default:
		return "unknown"
	}
}

// Frame types for multiplexing streams over a single WebSocket connection.
const (
	frameStream   byte = 0x01 // Stream data: [streamID uint32][payload]
	frameDatagram byte = 0x02 // Datagram: [payload]
	frameOpen     byte = 0x03 // Open stream: [streamID uint32]
	frameClose    byte = 0x04 // Close stream: [streamID uint32]
)

// WSConn wraps a WebSocket connection and provides stream multiplexing.
type WSConn struct {
	ws        *websocket.Conn
	mu        sync.Mutex
	streams   map[uint32]*WSStream
	nextID    uint32
	incoming  chan *WSStream
	datagrams chan []byte
	closed    bool
}

// NewWSConn wraps a websocket.Conn with multiplexing support.
func NewWSConn(ws *websocket.Conn) *WSConn {
	c := &WSConn{
		ws:        ws,
		streams:   make(map[uint32]*WSStream),
		incoming:  make(chan *WSStream, 16),
		datagrams: make(chan []byte, 64),
	}
	go c.readLoop()
	return c
}

func (c *WSConn) readLoop() {
	defer c.Close()
	for {
		var frame []byte
		if err := websocket.Message.Receive(c.ws, &frame); err != nil {
			return
		}
		if len(frame) < 1 {
			continue
		}

		switch frame[0] {
		case frameStream:
			if len(frame) < 5 {
				continue
			}
			streamID := binary.BigEndian.Uint32(frame[1:5])
			payload := frame[5:]

			c.mu.Lock()
			s, ok := c.streams[streamID]
			c.mu.Unlock()

			if ok {
				s.pushData(payload)
			}

		case frameDatagram:
			select {
			case c.datagrams <- frame[1:]:
			default:
				// Drop if buffer full (datagrams are unreliable anyway)
			}

		case frameOpen:
			if len(frame) < 5 {
				continue
			}
			streamID := binary.BigEndian.Uint32(frame[1:5])
			s := newWSStream(streamID, c)

			c.mu.Lock()
			c.streams[streamID] = s
			c.mu.Unlock()

			select {
			case c.incoming <- s:
			default:
			}

		case frameClose:
			if len(frame) < 5 {
				continue
			}
			streamID := binary.BigEndian.Uint32(frame[1:5])

			c.mu.Lock()
			if s, ok := c.streams[streamID]; ok {
				s.close()
				delete(c.streams, streamID)
			}
			c.mu.Unlock()
		}
	}
}

// AcceptStream waits for the next incoming stream from the client.
func (c *WSConn) AcceptStream() (*WSStream, error) {
	s, ok := <-c.incoming
	if !ok {
		return nil, fmt.Errorf("connection closed")
	}
	return s, nil
}

// OpenStream creates a new outbound stream.
func (c *WSConn) OpenStream() (*WSStream, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	s := newWSStream(id, c)
	c.streams[id] = s
	c.mu.Unlock()

	// Send open frame
	frame := make([]byte, 5)
	frame[0] = frameOpen
	binary.BigEndian.PutUint32(frame[1:], id)
	return s, c.send(frame)
}

// SendDatagram sends a datagram-like message (simulated over WebSocket).
func (c *WSConn) SendDatagram(data []byte) error {
	frame := make([]byte, 1+len(data))
	frame[0] = frameDatagram
	copy(frame[1:], data)
	return c.send(frame)
}

// ReceiveDatagram receives a datagram-like message.
func (c *WSConn) ReceiveDatagram() ([]byte, error) {
	data, ok := <-c.datagrams
	if !ok {
		return nil, fmt.Errorf("connection closed")
	}
	return data, nil
}

// Close closes the WebSocket connection and all streams.
func (c *WSConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	for _, s := range c.streams {
		s.close()
	}
	close(c.incoming)
	close(c.datagrams)
	c.mu.Unlock()
	return c.ws.Close()
}

func (c *WSConn) send(frame []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	return websocket.Message.Send(c.ws, frame)
}

func (c *WSConn) sendStreamData(streamID uint32, data []byte) error {
	frame := make([]byte, 5+len(data))
	frame[0] = frameStream
	binary.BigEndian.PutUint32(frame[1:], streamID)
	copy(frame[5:], data)
	return c.send(frame)
}

func (c *WSConn) sendStreamClose(streamID uint32) error {
	frame := make([]byte, 5)
	frame[0] = frameClose
	binary.BigEndian.PutUint32(frame[1:], streamID)
	return c.send(frame)
}

// WSStream is a multiplexed stream over a WebSocket connection.
type WSStream struct {
	id   uint32
	conn *WSConn
	buf  chan []byte
	done chan struct{}
	once sync.Once
}

func newWSStream(id uint32, conn *WSConn) *WSStream {
	return &WSStream{
		id:   id,
		conn: conn,
		buf:  make(chan []byte, 32),
		done: make(chan struct{}),
	}
}

func (s *WSStream) pushData(data []byte) {
	select {
	case s.buf <- data:
	case <-s.done:
	}
}

// Read reads data from the stream.
func (s *WSStream) Read(b []byte) (int, error) {
	select {
	case data, ok := <-s.buf:
		if !ok {
			return 0, io.EOF
		}
		n := copy(b, data)
		return n, nil
	case <-s.done:
		return 0, io.EOF
	}
}

// Write writes data to the stream.
func (s *WSStream) Write(b []byte) (int, error) {
	if err := s.conn.sendStreamData(s.id, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes the stream.
func (s *WSStream) Close() error {
	s.once.Do(func() {
		close(s.done)
	})
	return s.conn.sendStreamClose(s.id)
}

func (s *WSStream) close() {
	s.once.Do(func() {
		close(s.done)
	})
}

// ID returns the stream ID.
func (s *WSStream) ID() uint32 {
	return s.id
}

// Handler returns an http.Handler that accepts WebSocket connections
// and provides the same multiplexed stream/datagram API.
// Use this alongside the WebTransport server to serve fallback clients.
func Handler(onConnect func(*WSConn)) http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		conn := NewWSConn(ws)
		defer conn.Close()
		onConnect(conn)
	})
}
