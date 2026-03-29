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

func TestHandleBothIntegration(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/both", HandleBoth(
		func(s *Stream, c *Context) {
			defer s.Close()
			msg, _ := s.ReadMessage()
			s.WriteMessage(append([]byte("stream:"), msg...))
		},
		func(data []byte, c *Context) []byte {
			return append([]byte("dgram:"), data...)
		},
	))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/both", addr), nil)
	defer session.CloseWithError(0, "")

	// Test datagram
	session.SendDatagram([]byte("ping"))
	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("datagram: %v", err)
	}
	if string(reply) != "dgram:ping" {
		t.Errorf("datagram: expected 'dgram:ping', got %q", reply)
	}

	// Test stream
	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}
	s.WriteMessage([]byte("hello"))
	streamReply, _ := s.ReadMessage()
	if string(streamReply) != "stream:hello" {
		t.Errorf("stream: expected 'stream:hello', got %q", streamReply)
	}
}
