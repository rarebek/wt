// Example: RPC service over WebTransport.
// Demonstrates using RPCServer for a calculator service.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

func main() {
	server := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	log.Printf("cert: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())

	rpc := wt.NewRPCServer()

	rpc.Register("add", func(params json.RawMessage) (any, error) {
		var args [2]float64
		json.Unmarshal(params, &args)
		return args[0] + args[1], nil
	})

	rpc.Register("multiply", func(params json.RawMessage) (any, error) {
		var args [2]float64
		json.Unmarshal(params, &args)
		return args[0] * args[1], nil
	})

	rpc.Register("echo", func(params json.RawMessage) (any, error) {
		var msg string
		json.Unmarshal(params, &msg)
		return fmt.Sprintf("echo: %s", msg), nil
	})

	server.Handle("/rpc", wt.HandleStream(func(s *wt.Stream, c *wt.Context) {
		rpc.Serve(s)
	}))

	log.Printf("RPC service on %s", server.Addr())
	log.Fatal(server.ListenAndServe())
}
