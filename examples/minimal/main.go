// The simplest possible wt server — 15 lines.
package main

import (
	"log"

	"github.com/rarebek/wt"
)

func main() {
	s := wt.New(wt.WithAddr(":4433"), wt.WithSelfSignedTLS())
	log.Printf("cert: %s", s.CertHash())
	s.Handle("/echo", wt.HandleDatagram(func(d []byte, _ *wt.Context) []byte { return d }))
	log.Fatal(s.ListenAndServe())
}
