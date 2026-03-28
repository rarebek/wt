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

func TestBackpressureWriterDropsE2E(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	resultCh := make(chan [2]uint64, 1)

	server.Handle("/bp", HandleDatagram(func(data []byte, c *Context) []byte {
		// Use backpressure writer with tiny buffer for datagrams
		bw := &BackpressureWriter{
			queue:  make(chan []byte, 2), // very small buffer
			done:   make(chan struct{}),
		}

		// Try to send 10 messages instantly — most should be dropped
		for i := range 10 {
			bw.Send([]byte(fmt.Sprintf("msg-%d", i)))
		}

		sent, dropped := bw.Stats()
		resultCh <- [2]uint64{sent, dropped}
		bw.Close()

		return []byte("ok")
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/bp", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Trigger the handler
	session.SendDatagram([]byte("go"))

	select {
	case result := <-resultCh:
		sent, dropped := result[0], result[1]
		t.Logf("backpressure: sent=%d dropped=%d (buffer=2, attempted=10)", sent, dropped)
		if dropped == 0 {
			t.Error("expected some messages to be dropped with buffer size 2")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
