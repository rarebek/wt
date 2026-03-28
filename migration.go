package wt

// This file documents how QUIC connection migration works with the wt framework.
// Connection migration is handled transparently by quic-go — the framework
// doesn't need to do anything special.
//
// # How It Works
//
// QUIC connections use Connection IDs instead of IP:port tuples.
// When a mobile device switches from WiFi to cellular, the IP address changes
// but the Connection ID stays the same. quic-go automatically handles this.
//
// From the framework's perspective:
//   - Sessions survive network changes without reconnection
//   - Streams and datagrams continue working after migration
//   - Context.RemoteAddr() may return a new address after migration
//   - No session state is lost
//
// # Server-Side (Automatic)
//
// The server automatically detects NAT rebinding (same client, new IP/port)
// and performs path validation. No framework code needed.
//
// # Client-Side (via quic-go API)
//
// For Go clients that need to explicitly migrate (e.g., switch from WiFi to cellular):
//
//	import "github.com/quic-go/quic-go"
//
//	// 1. Create a new transport for the new network interface
//	newConn, _ := net.ListenUDP("udp", &net.UDPAddr{})
//	newTransport := &quic.Transport{Conn: newConn}
//
//	// 2. Add a path through the underlying QUIC connection
//	// (requires access to the raw quic.Conn via the client SDK)
//	path, _ := quicConn.AddPath(newTransport)
//
//	// 3. Probe the new path
//	ctx, cancel := context.WithTimeout(ctx, time.Second)
//	path.Probe(ctx)
//	cancel()
//
//	// 4. Switch to the new path
//	path.Switch()
//
//	// 5. Close the old path
//	path.Close()
//
// # What the Framework Handles Automatically
//
//   - Session identity preservation across migrations
//   - Room membership maintained (no rejoin needed)
//   - Context store values preserved
//   - Active streams continue uninterrupted
//   - Middleware state preserved
//
// # Limitations
//
//   - Only client-initiated migration (per RFC 9000)
//   - Server cannot force a client to migrate
//   - quic-go does not yet support Preferred Address (issue #4965)
