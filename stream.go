package wt

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/webtransport-go"
)

// Stream wraps a bidirectional WebTransport stream with framing and helpers.
type Stream struct {
	raw    *webtransport.Stream
	ctx    *Context
	hdrBuf [4]byte // reusable header buffer for ReadMessage
}

func newStream(s *webtransport.Stream, ctx *Context) *Stream {
	return &Stream{raw: s, ctx: ctx}
}

// Raw returns the underlying webtransport.Stream.
func (s *Stream) Raw() *webtransport.Stream {
	return s.raw
}

// Context returns the session context this stream belongs to.
func (s *Stream) SessionContext() *Context {
	return s.ctx
}

// Read reads raw bytes from the stream.
func (s *Stream) Read(b []byte) (int, error) {
	return s.raw.Read(b)
}

// Write writes raw bytes to the stream.
func (s *Stream) Write(b []byte) (int, error) {
	return s.raw.Write(b)
}

// Close closes the stream.
func (s *Stream) Close() error {
	return s.raw.Close()
}

// SetDeadline sets read and write deadlines.
func (s *Stream) SetDeadline(t time.Time) error {
	return s.raw.SetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.raw.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.raw.SetWriteDeadline(t)
}

// WriteMessage writes a length-prefixed message to the stream.
// Format: [4 bytes big-endian length][payload]
func (s *Stream) WriteMessage(data []byte) error {
	if len(data) > MaxMessageSize {
		return fmt.Errorf("wt: message too large: %d > %d", len(data), MaxMessageSize)
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := s.raw.Write(header); err != nil {
		return err
	}
	_, err := s.raw.Write(data)
	return err
}

// ReadMessage reads a length-prefixed message from the stream.
func (s *Stream) ReadMessage() ([]byte, error) {
	if _, err := io.ReadFull(s.raw, s.hdrBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(s.hdrBuf[:])
	if length > uint32(MaxMessageSize) {
		return nil, fmt.Errorf("wt: message too large: %d > %d", length, MaxMessageSize)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(s.raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

// CancelRead cancels the read side of the stream.
func (s *Stream) CancelRead(code uint32) {
	s.raw.CancelRead(webtransport.StreamErrorCode(code))
}

// CancelWrite cancels the write side of the stream.
func (s *Stream) CancelWrite(code uint32) {
	s.raw.CancelWrite(webtransport.StreamErrorCode(code))
}

// MaxMessageSize is the maximum size of a length-prefixed message (16 MB).
const MaxMessageSize = 16 * 1024 * 1024

// SendStream wraps a unidirectional send stream.
type SendStream struct {
	raw *webtransport.SendStream
}

// Write writes bytes to the send stream.
func (s *SendStream) Write(b []byte) (int, error) {
	return s.raw.Write(b)
}

// WriteMessage writes a length-prefixed message.
func (s *SendStream) WriteMessage(data []byte) error {
	if len(data) > MaxMessageSize {
		return fmt.Errorf("wt: message too large: %d > %d", len(data), MaxMessageSize)
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := s.raw.Write(header); err != nil {
		return err
	}
	_, err := s.raw.Write(data)
	return err
}

// Close closes the send stream.
func (s *SendStream) Close() error {
	return s.raw.Close()
}

// SetWriteDeadline sets the write deadline.
func (s *SendStream) SetWriteDeadline(t time.Time) error {
	return s.raw.SetWriteDeadline(t)
}

// CancelWrite cancels writing.
func (s *SendStream) CancelWrite(code uint32) {
	s.raw.CancelWrite(webtransport.StreamErrorCode(code))
}

// ReceiveStream wraps a unidirectional receive stream.
type ReceiveStream struct {
	raw *webtransport.ReceiveStream
}

// Read reads bytes from the receive stream.
func (s *ReceiveStream) Read(b []byte) (int, error) {
	return s.raw.Read(b)
}

// ReadMessage reads a length-prefixed message.
func (s *ReceiveStream) ReadMessage() ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(s.raw, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length > uint32(MaxMessageSize) {
		return nil, fmt.Errorf("wt: message too large: %d > %d", length, MaxMessageSize)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(s.raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

// SetReadDeadline sets the read deadline.
func (s *ReceiveStream) SetReadDeadline(t time.Time) error {
	return s.raw.SetReadDeadline(t)
}

// CancelRead cancels reading.
func (s *ReceiveStream) CancelRead(code uint32) {
	s.raw.CancelRead(webtransport.StreamErrorCode(code))
}
