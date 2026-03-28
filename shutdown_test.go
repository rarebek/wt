package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func TestGracefulShutdown(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	sessionActive := make(chan struct{})
	sessionDone := make(chan struct{})

	server.Handle("/drain", func(c *Context) {
		close(sessionActive) // signal that session is up
		<-c.Context().Done() // wait for session close
		close(sessionDone)
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	// Connect a client
	ctx := context.Background()
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/drain", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Wait for session to be active
	select {
	case <-sessionActive:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for session")
	}

	if server.SessionCount() != 1 {
		t.Errorf("expected 1 session, got %d", server.SessionCount())
	}

	// Graceful shutdown with 2s timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go server.Shutdown(shutdownCtx)

	// Session should be force-closed within the timeout
	select {
	case <-sessionDone:
		// Session was closed by shutdown
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for session to drain")
	}

	session.CloseWithError(0, "")
}
