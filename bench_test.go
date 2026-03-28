package wt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
)

func BenchmarkRouterMatch(b *testing.B) {
	r := NewRouter()
	r.Add("/chat/{room}", func(c *Context) {})
	r.Add("/game/{id}/input", func(c *Context) {})
	r.Add("/game/{id}/state", func(c *Context) {})
	r.Add("/user/{org}/{team}/{id}", func(c *Context) {})
	r.Add("/static/assets", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/game/42/input")
	}
}

func BenchmarkRouterMatchDeep(b *testing.B) {
	r := NewRouter()
	r.Add("/a/{b}/{c}/{d}/{e}", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/a/1/2/3/4")
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	noop := func(c *Context) {}

	mw1 := MiddlewareFunc(func(c *Context, next HandlerFunc) { next(c) })
	mw2 := MiddlewareFunc(func(c *Context, next HandlerFunc) { next(c) })
	mw3 := MiddlewareFunc(func(c *Context, next HandlerFunc) { next(c) })

	chain := buildChain(noop, []MiddlewareFunc{mw1, mw2, mw3})

	b.ResetTimer()
	for b.Loop() {
		chain(nil)
	}
}

func BenchmarkSessionStoreAdd(b *testing.B) {
	ss := NewSessionStore()
	contexts := make([]*Context, b.N)
	for i := range contexts {
		contexts[i] = &Context{
			id:    fmt.Sprintf("session-%d", i),
			store: make(map[string]any),
		}
	}

	b.ResetTimer()
	for i := range b.N {
		ss.Add(contexts[i])
	}
}

func BenchmarkSessionStoreGet(b *testing.B) {
	ss := NewSessionStore()
	for i := range 1000 {
		ctx := &Context{
			id:    fmt.Sprintf("session-%d", i),
			store: make(map[string]any),
		}
		ss.Add(ctx)
	}

	b.ResetTimer()
	for b.Loop() {
		ss.Get("session-500")
	}
}

func BenchmarkRoomMembers(b *testing.B) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("bench")

	for i := range 100 {
		ctx := &Context{
			id:    fmt.Sprintf("member-%d", i),
			store: make(map[string]any),
		}
		room.Join(ctx)
	}

	b.ResetTimer()
	for b.Loop() {
		room.Members() // allocates slice copy
	}
}

func BenchmarkRoomForEach(b *testing.B) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("bench")

	for i := range 100 {
		ctx := &Context{
			id:    fmt.Sprintf("member-%d", i),
			store: make(map[string]any),
		}
		room.Join(ctx)
	}

	b.ResetTimer()
	for b.Loop() {
		room.ForEach(func(c *Context) {
			_ = c // zero-alloc iteration
		})
	}
}

func BenchmarkContextSetGet(b *testing.B) {
	ctx := &Context{
		store: make(map[string]any),
	}
	ctx.Set("user", "alice")

	b.ResetTimer()
	for b.Loop() {
		ctx.Get("user")
	}
}

func BenchmarkContextSetGetParallel(b *testing.B) {
	ctx := &Context{
		store: make(map[string]any),
	}
	ctx.Set("user", "alice")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx.Get("user")
		}
	})
}

func BenchmarkEventBusEmit(b *testing.B) {
	bus := NewEventBus()
	bus.On(EventConnect, func(e Event) {})
	bus.On(EventConnect, func(e Event) {})
	bus.On(EventConnect, func(e Event) {})

	event := Event{Type: EventConnect}

	b.ResetTimer()
	for b.Loop() {
		bus.Emit(event)
	}
}

func BenchmarkResumeStoreSaveRestore(b *testing.B) {
	rs := NewResumeStore(5 * time.Minute)
	ctx := &Context{
		store: map[string]any{"user": "alice", "role": "admin"},
	}

	b.ResetTimer()
	for b.Loop() {
		token := rs.Save(ctx)
		newCtx := &Context{store: make(map[string]any)}
		rs.Restore(newCtx, token)
	}
}

// BenchmarkStreamEchoE2E benchmarks end-to-end message echo over real QUIC.
func BenchmarkStreamEchoE2E(b *testing.B) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		b.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var wg sync.WaitGroup
	server.Handle("/bench", func(c *Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer stream.Close()
				for {
					msg, err := stream.ReadMessage()
					if err != nil {
						return
					}
					if err := stream.WriteMessage(msg); err != nil {
						return
					}
				}
			}()
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx := context.Background()
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/bench", addr), nil)
	if err != nil {
		b.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		b.Fatalf("open stream error: %v", err)
	}
	s := &Stream{raw: stream, ctx: nil}

	msg := make([]byte, 64)
	for i := range msg {
		msg[i] = byte(i)
	}

	b.ResetTimer()
	b.SetBytes(int64(len(msg)))

	for range b.N {
		if err := s.WriteMessage(msg); err != nil {
			b.Fatal(err)
		}
		if _, err := s.ReadMessage(); err != nil {
			b.Fatal(err)
		}
	}
}
