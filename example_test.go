package wt_test

import (
	"fmt"
	"io"
	"log"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/codec"
	"github.com/rarebek/wt/middleware"
)

func ExampleNew() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	server.Handle("/echo", func(c *wt.Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				io.Copy(stream, stream)
			}()
		}
	})

	fmt.Println("Server configured on", server.Addr())
	// Output: Server configured on :4433
}

func ExampleServer_Handle() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

	// Simple echo handler
	server.Handle("/echo", func(c *wt.Context) {
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				msg, _ := stream.ReadMessage()
				stream.WriteMessage(msg)
			}()
		}
	})

	fmt.Println("Handler registered")
	// Output: Handler registered
}

func ExampleServer_Group() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

	// Create a group with auth middleware
	api := server.Group("/api", func(c *wt.Context, next wt.HandlerFunc) {
		// Check auth header
		if c.Request().Header.Get("Authorization") == "" {
			c.CloseWithError(401, "unauthorized")
			return
		}
		next(c)
	})

	api.Handle("/data", func(c *wt.Context) {
		// Only authenticated users reach here
		_ = c
	})

	fmt.Println("Group registered")
	// Output: Group registered
}

func ExampleHandleStream() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

	// HandleStream auto-accepts streams and calls handler for each
	server.Handle("/echo", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(msg)
	}))

	fmt.Println("Stream handler registered")
	// Output: Stream handler registered
}

func ExampleHandleDatagram() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

	// HandleDatagram auto-receives datagrams and echoes reply
	server.Handle("/ping", wt.HandleDatagram(func(data []byte, c *wt.Context) []byte {
		return append([]byte("pong:"), data...)
	}))

	fmt.Println("Datagram handler registered")
	// Output: Datagram handler registered
}

func ExampleNewStreamMux() {
	mux := wt.NewStreamMux()

	const (
		TypeChat  uint16 = 1
		TypeGame  uint16 = 2
	)

	mux.Handle(TypeChat, func(s *wt.Stream, c *wt.Context) {
		// Handle chat stream
	})
	mux.Handle(TypeGame, func(s *wt.Stream, c *wt.Context) {
		// Handle game stream
	})

	fmt.Println("StreamMux configured with 2 handlers")
	// Output: StreamMux configured with 2 handlers
}

func ExampleNewTypedStream() {
	type Input struct {
		Action string `json:"action"`
	}
	type Output struct {
		Result string `json:"result"`
	}

	// In a real handler:
	// stream, _ := c.AcceptStream()
	// typed := wt.NewTypedStream[Input, Output](stream, codec.JSON{})
	// input, _ := typed.Read()
	// typed.Write(Output{Result: "ok"})

	_ = codec.JSON{}
	fmt.Println("TypedStream example")
	// Output: TypedStream example
}

func ExampleNewRoomManager() {
	rooms := wt.NewRoomManager()

	lobby := rooms.GetOrCreate("lobby")
	lobby.OnJoin(func(c *wt.Context) {
		log.Printf("user %s joined lobby", c.ID())
	})

	fmt.Println("Room manager created")
	// Output: Room manager created
}

func Example_middleware() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())

	// Stack middleware
	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Use(middleware.RateLimit(100))

	server.Handle("/app", func(c *wt.Context) {
		_ = c
	})

	fmt.Println("Middleware configured")
	// Output: Middleware configured
}
