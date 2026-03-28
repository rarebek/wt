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

func TestStreamsIterator(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	streamCount := 0
	server.Handle("/iter", func(c *Context) {
		for stream := range Streams(c) {
			streamCount++
			go func() {
				defer stream.Close()
				msg, _ := stream.ReadMessage()
				stream.WriteMessage(msg)
			}()
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/iter", addr), nil)
	defer session.CloseWithError(0, "")

	// Open 3 streams via the iterator
	for i := range 3 {
		raw, _ := session.OpenStreamSync(ctx)
		s := &Stream{raw: raw}
		s.WriteMessage([]byte(fmt.Sprintf("iter-%d", i)))
		reply, _ := s.ReadMessage()
		if string(reply) != fmt.Sprintf("iter-%d", i) {
			t.Errorf("stream %d: expected 'iter-%d', got %q", i, i, reply)
		}
	}

	session.CloseWithError(0, "")
	time.Sleep(100 * time.Millisecond)

	if streamCount != 3 {
		t.Errorf("expected 3 streams via iterator, got %d", streamCount)
	}
}
