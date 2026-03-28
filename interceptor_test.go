package wt

import (
	"fmt"
	"testing"
)

func TestInterceptorOnRead(t *testing.T) {
	readCalled := false
	si := &StreamInterceptor{
		onRead: func(data []byte) ([]byte, error) {
			readCalled = true
			return append([]byte("prefix:"), data...), nil
		},
	}

	result, err := si.onRead([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readCalled {
		t.Error("onRead was not called")
	}
	if string(result) != "prefix:hello" {
		t.Errorf("expected 'prefix:hello', got %q", result)
	}
}

func TestInterceptorOnWrite(t *testing.T) {
	si := &StreamInterceptor{
		onWrite: func(data []byte) ([]byte, error) {
			if len(data) > 100 {
				return nil, fmt.Errorf("message too large")
			}
			return data, nil
		},
	}

	// Valid message
	result, err := si.onWrite([]byte("ok"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}

	// Too large
	_, err = si.onWrite(make([]byte, 200))
	if err == nil {
		t.Error("expected error for large message")
	}
}

func TestInterceptNil(t *testing.T) {
	// No interceptors set — should pass through
	si := &StreamInterceptor{}

	if si.onRead != nil {
		t.Error("onRead should be nil")
	}
	if si.onWrite != nil {
		t.Error("onWrite should be nil")
	}
}
