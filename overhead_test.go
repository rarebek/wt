package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// BenchmarkRawWebtransportEcho benchmarks raw quic-go/webtransport-go
// WITHOUT our framework — direct Session.AcceptStream + io.Copy.
// This measures the protocol baseline.
func BenchmarkRawWebtransportEcho(b *testing.B) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Raw server — no wt framework
	cert, _ := generateSelfSigned()
	mux := http.NewServeMux()
	wtServer := &webtransport.Server{
		H3: &http3.Server{
			Addr:      addr,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{*cert}, NextProtos: []string{"h3"}},
			Handler:   mux,
		},
	}
	webtransport.ConfigureHTTP3Server(wtServer.H3)

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		session, err := wtServer.Upgrade(w, r)
		if err != nil {
			return
		}
		for {
			stream, err := session.AcceptStream(context.Background())
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				io.Copy(stream, stream) // raw echo, no framing
			}()
		}
	})

	conn, _ := net.ListenPacket("udp", addr)
	go wtServer.Serve(conn)
	time.Sleep(100 * time.Millisecond)
	defer wtServer.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	defer session.CloseWithError(0, "")

	raw, _ := session.OpenStreamSync(ctx)
	msg := make([]byte, 64)
	buf := make([]byte, 64)

	b.ResetTimer()
	b.SetBytes(64)
	for range b.N {
		raw.Write(msg)
		io.ReadFull(raw, buf)
	}
}

// BenchmarkFrameworkEcho benchmarks our wt framework with the same workload.
// Includes: routing, context creation, session tracking, message framing.
func BenchmarkFrameworkEcho(b *testing.B) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())
	server.Handle("/echo", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			if err := s.WriteMessage(msg); err != nil {
				return
			}
		}
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	defer session.CloseWithError(0, "")

	raw, _ := session.OpenStreamSync(ctx)
	stream := &Stream{raw: raw}
	msg := make([]byte, 64)

	b.ResetTimer()
	b.SetBytes(64)
	for range b.N {
		stream.WriteMessage(msg)
		stream.ReadMessage()
	}
}
