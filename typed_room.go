package wt

import (
	"github.com/rarebek/wt/codec"
)

// TypedRoom provides type-safe broadcast over a Room using a codec.
type TypedRoom[T any] struct {
	room  *Room
	codec codec.Codec
}

// NewTypedRoom wraps a Room with typed broadcast support.
func NewTypedRoom[T any](room *Room, c codec.Codec) *TypedRoom[T] {
	return &TypedRoom[T]{room: room, codec: c}
}

// Broadcast encodes and broadcasts a value to all room members via datagrams.
func (tr *TypedRoom[T]) Broadcast(v T) error {
	data, err := tr.codec.Marshal(v)
	if err != nil {
		return err
	}
	tr.room.Broadcast(data)
	return nil
}

// BroadcastExcept broadcasts to all members except the specified session.
func (tr *TypedRoom[T]) BroadcastExcept(v T, excludeID string) error {
	data, err := tr.codec.Marshal(v)
	if err != nil {
		return err
	}
	tr.room.BroadcastExcept(data, excludeID)
	return nil
}

// Room returns the underlying Room.
func (tr *TypedRoom[T]) Room() *Room {
	return tr.room
}
