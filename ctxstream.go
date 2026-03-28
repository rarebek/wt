package wt

import (
	"context"
	"time"
)

// ContextStream wraps a Stream with context-aware read/write operations.
// When the context is cancelled, all pending reads and writes are unblocked.
type ContextStream struct {
	*Stream
	ctx    context.Context
	cancel context.CancelFunc
}

// WithContext creates a ContextStream that respects the given context.
// When ctx is cancelled, the stream is closed automatically.
func (s *Stream) WithContext(ctx context.Context) *ContextStream {
	ctx, cancel := context.WithCancel(ctx)
	cs := &ContextStream{
		Stream: s,
		ctx:    ctx,
		cancel: cancel,
	}
	// Auto-close stream when context is cancelled
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return cs
}

// Context returns the stream's context.
func (cs *ContextStream) Context() context.Context {
	return cs.ctx
}

// ReadMessageContext reads a length-prefixed message with context support.
// Returns context.Canceled or context.DeadlineExceeded if the context is done.
func (cs *ContextStream) ReadMessageContext() ([]byte, error) {
	// Set deadline from context if available
	if deadline, ok := cs.ctx.Deadline(); ok {
		cs.Stream.SetReadDeadline(deadline)
	}

	msg, err := cs.Stream.ReadMessage()
	if err != nil {
		// Check if the error is due to context cancellation
		if cs.ctx.Err() != nil {
			return nil, cs.ctx.Err()
		}
		return nil, err
	}
	return msg, nil
}

// WriteMessageContext writes a length-prefixed message with context support.
func (cs *ContextStream) WriteMessageContext(data []byte) error {
	if deadline, ok := cs.ctx.Deadline(); ok {
		cs.Stream.SetWriteDeadline(deadline)
	}

	if err := cs.Stream.WriteMessage(data); err != nil {
		if cs.ctx.Err() != nil {
			return cs.ctx.Err()
		}
		return err
	}
	return nil
}

// Close cancels the context and closes the stream.
func (cs *ContextStream) Close() error {
	cs.cancel()
	return cs.Stream.Close()
}

// WithDeadline creates a stream that automatically closes at the given deadline.
func (s *Stream) WithDeadline(deadline time.Time) *ContextStream {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	cs := &ContextStream{
		Stream: s,
		ctx:    ctx,
		cancel: cancel,
	}
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return cs
}

// WithTimeout creates a stream that automatically closes after the given duration.
func (s *Stream) WithTimeout(d time.Duration) *ContextStream {
	return s.WithDeadline(time.Now().Add(d))
}
