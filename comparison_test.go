package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
	"golang.org/x/net/websocket"
)

// ============================================================
// REAL COMPARISON: WebTransport (QUIC/UDP) vs WebSocket (TCP)
// Same workload, same machine, actual measurements.
// No marketing. Just numbers.
// ============================================================

// --- Single-stream echo latency (64 bytes) ---

func BenchmarkWT_Echo_64B(b *testing.B) {
	s, addr := startWTBench(b)
	defer s.Close()

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

func BenchmarkWS_Echo_64B(b *testing.B) {
	ws := startWSBench(b)
	defer ws.Close()

	msg := make([]byte, 64)
	buf := make([]byte, 128)

	b.ResetTimer()
	b.SetBytes(64)
	for range b.N {
		ws.Write(msg)
		ws.Read(buf)
	}
}

// --- Single-stream echo (1 KB) ---

func BenchmarkWT_Echo_1KB(b *testing.B) {
	s, addr := startWTBench(b)
	defer s.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	defer session.CloseWithError(0, "")

	raw, _ := session.OpenStreamSync(ctx)
	stream := &Stream{raw: raw}
	msg := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	for range b.N {
		stream.WriteMessage(msg)
		stream.ReadMessage()
	}
}

func BenchmarkWS_Echo_1KB(b *testing.B) {
	ws := startWSBench(b)
	defer ws.Close()

	msg := make([]byte, 1024)
	buf := make([]byte, 2048)

	b.ResetTimer()
	b.SetBytes(1024)
	for range b.N {
		ws.Write(msg)
		ws.Read(buf)
	}
}

// --- 5 parallel data channels (THE key difference) ---

// WebTransport: 5 independent QUIC streams. No head-of-line blocking.
func BenchmarkWT_5Parallel_64B(b *testing.B) {
	s, addr := startWTBench(b)
	defer s.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	defer session.CloseWithError(0, "")

	streams := make([]*Stream, 5)
	for i := range streams {
		raw, _ := session.OpenStreamSync(ctx)
		streams[i] = &Stream{raw: raw}
	}

	msg := make([]byte, 64)
	b.ResetTimer()
	b.SetBytes(64 * 5)

	for range b.N {
		var wg sync.WaitGroup
		for _, st := range streams {
			wg.Add(1)
			go func(s *Stream) {
				defer wg.Done()
				s.WriteMessage(msg)
				s.ReadMessage()
			}(st)
		}
		wg.Wait()
	}
}

// WebSocket: must serialize 5 messages on one pipe. If msg 2 is slow, 3-4-5 wait.
func BenchmarkWS_5Serial_64B(b *testing.B) {
	ws := startWSBench(b)
	defer ws.Close()

	msg := make([]byte, 64)
	buf := make([]byte, 128)

	b.ResetTimer()
	b.SetBytes(64 * 5)

	for range b.N {
		for range 5 {
			ws.Write(msg)
			ws.Read(buf)
		}
	}
}

// --- Datagram echo (WebTransport ONLY — no WebSocket equivalent) ---

func BenchmarkWT_Datagram_64B(b *testing.B) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	server := New(WithAddr(addr), WithSelfSignedTLS())
	server.Handle("/dg", func(c *Context) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			c.SendDatagram(data)
		}
	})
	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/dg", addr), nil)
	defer session.CloseWithError(0, "")

	msg := make([]byte, 64)
	b.ResetTimer()
	b.SetBytes(64)
	for range b.N {
		session.SendDatagram(msg)
		session.ReceiveDatagram(ctx)
	}
}

// --- Helpers ---

func startWTBench(b *testing.B) (*Server, string) {
	b.Helper()
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
	return server, addr
}

func startWSBench(b *testing.B) *websocket.Conn {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		websocket.Handler(func(ws *websocket.Conn) {
			buf := make([]byte, 4096)
			for {
				n, err := ws.Read(buf)
				if err != nil {
					return
				}
				ws.Write(buf[:n])
			}
		}).ServeHTTP(w, r)
	}))

	b.Cleanup(func() { srv.Close() })

	wsURL := "ws" + srv.URL[len("http"):]
	ws, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		b.Fatal(err)
	}
	return ws
}
