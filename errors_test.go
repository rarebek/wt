package wt

import (
	"fmt"
	"testing"
)

func TestSessionCloseError(t *testing.T) {
	err := &SessionCloseError{Code: 401, Message: "unauthorized"}

	if err.Error() != "wt: session closed with code 401: unauthorized" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !IsSessionClosed(err) {
		t.Error("expected IsSessionClosed to return true")
	}
}

func TestStreamCloseError(t *testing.T) {
	err := &StreamCloseError{Code: 100, Remote: true}
	if err.Error() != "wt: stream closed by remote with code 100" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	err2 := &StreamCloseError{Code: 0, Remote: false}
	if err2.Error() != "wt: stream closed by local with code 0" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}

	if !IsStreamClosed(err) {
		t.Error("expected IsStreamClosed to return true")
	}
}

func TestIsNotCloseError(t *testing.T) {
	if IsSessionClosed(nil) {
		t.Error("nil should not be a session close error")
	}
	if IsStreamClosed(nil) {
		t.Error("nil should not be a stream close error")
	}
}

func TestConnectionError(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := &ConnectionError{Op: "dial", Addr: "localhost:4433", Wrapped: inner}

	if !IsConnectionError(err) {
		t.Error("expected IsConnectionError true")
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestUpgradeError(t *testing.T) {
	err := &UpgradeError{StatusCode: 500, Message: "internal error"}
	if !IsUpgradeError(err) {
		t.Error("expected IsUpgradeError true")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestMessageError(t *testing.T) {
	inner := fmt.Errorf("broken pipe")
	err := &MessageError{Op: "write", Size: 1024, Wrapped: inner}
	if !IsMessageError(err) {
		t.Error("expected IsMessageError true")
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestErrorCodes(t *testing.T) {
	if CodeOK != 0 {
		t.Error("CodeOK should be 0")
	}
	if CodeUnauthorized != 401 {
		t.Error("CodeUnauthorized should be 401")
	}
	if CodeProtocolError != 0x1000 {
		t.Error("CodeProtocolError should be 0x1000")
	}
}
