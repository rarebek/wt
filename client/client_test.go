package client

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := New("https://localhost:4433/echo")

	if c.url != "https://localhost:4433/echo" {
		t.Errorf("expected url to be set, got %q", c.url)
	}
	if c.reconnect {
		t.Error("reconnect should be false by default")
	}
	if c.codec.Name() != "json" {
		t.Errorf("expected default codec 'json', got %q", c.codec.Name())
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c := New("https://example.com/test",
		WithReconnect(1*time.Second, 30*time.Second),
	)

	if !c.reconnect {
		t.Error("expected reconnect to be true")
	}
	if c.reconnectMin != 1*time.Second {
		t.Errorf("expected reconnectMin 1s, got %v", c.reconnectMin)
	}
	if c.reconnectMax != 30*time.Second {
		t.Errorf("expected reconnectMax 30s, got %v", c.reconnectMax)
	}
}

func TestClientNotConnected(t *testing.T) {
	c := New("https://localhost:4433/echo")

	if c.Session() != nil {
		t.Error("session should be nil before connecting")
	}

	_, err := c.OpenStream(context.TODO())
	if err == nil {
		t.Error("expected error when not connected")
	}

	_, err = c.AcceptStream(context.TODO())
	if err == nil {
		t.Error("expected error when not connected")
	}

	err = c.SendDatagram([]byte("test"))
	if err == nil {
		t.Error("expected error when not connected")
	}

	_, err = c.ReceiveDatagram(context.TODO())
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestClientClose(t *testing.T) {
	c := New("https://localhost:4433/echo")

	// Close without connecting should not error
	err := c.Close()
	if err != nil {
		t.Errorf("close on unconnected client should not error: %v", err)
	}
}
