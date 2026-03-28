package wt

import (
	"github.com/rarebek/wt/codec"
)

// TypedStream provides type-safe read/write over a Stream using a codec.
type TypedStream[R any, W any] struct {
	stream *Stream
	codec  codec.Codec
}

// NewTypedStream wraps a Stream with typed encoding/decoding.
func NewTypedStream[R any, W any](s *Stream, c codec.Codec) *TypedStream[R, W] {
	return &TypedStream[R, W]{stream: s, codec: c}
}

// Read reads and decodes a message from the stream.
func (ts *TypedStream[R, W]) Read() (R, error) {
	var zero R
	data, err := ts.stream.ReadMessage()
	if err != nil {
		return zero, err
	}
	var v R
	if err := ts.codec.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}

// Write encodes and writes a message to the stream.
func (ts *TypedStream[R, W]) Write(v W) error {
	data, err := ts.codec.Marshal(v)
	if err != nil {
		return err
	}
	return ts.stream.WriteMessage(data)
}

// Stream returns the underlying Stream.
func (ts *TypedStream[R, W]) Stream() *Stream {
	return ts.stream
}

// Close closes the underlying stream.
func (ts *TypedStream[R, W]) Close() error {
	return ts.stream.Close()
}

// TypedDatagram provides type-safe datagram read/write on a Context.
type TypedDatagram[T any] struct {
	ctx   *Context
	codec codec.Codec
}

// NewTypedDatagram wraps a Context with typed datagram encoding/decoding.
func NewTypedDatagram[T any](ctx *Context, c codec.Codec) *TypedDatagram[T] {
	return &TypedDatagram[T]{ctx: ctx, codec: c}
}

// Send encodes and sends a datagram.
func (td *TypedDatagram[T]) Send(v T) error {
	data, err := td.codec.Marshal(v)
	if err != nil {
		return err
	}
	return td.ctx.SendDatagram(data)
}

// Receive receives and decodes a datagram.
func (td *TypedDatagram[T]) Receive() (T, error) {
	var zero T
	data, err := td.ctx.ReceiveDatagram()
	if err != nil {
		return zero, err
	}
	var v T
	if err := td.codec.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}
