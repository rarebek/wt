// Command wtbench is a load testing tool for WebTransport servers.
//
// Usage:
//
//	go run ./wtbench -url https://localhost:4433/echo -clients 100 -streams 10 -duration 30s
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/webtransport-go"
)

func main() {
	url := flag.String("url", "https://localhost:4433/echo", "WebTransport server URL")
	clients := flag.Int("clients", 10, "Number of concurrent clients")
	streams := flag.Int("streams", 5, "Streams per client")
	duration := flag.Duration("duration", 10*time.Second, "Test duration")
	msgSize := flag.Int("msg-size", 64, "Message size in bytes")
	insecure := flag.Bool("insecure", true, "Skip TLS verification")
	flag.Parse()

	log.Printf("wtbench: %d clients, %d streams/client, %d byte msgs, %v duration",
		*clients, *streams, *msgSize, *duration)
	log.Printf("target: %s", *url)

	ctx, cancel := context.WithTimeout(context.Background(), *duration+10*time.Second)
	defer cancel()

	var (
		totalMessages atomic.Int64
		totalBytes    atomic.Int64
		totalErrors   atomic.Int64
		totalLatency  atomic.Int64 // nanoseconds
	)

	start := time.Now()
	var wg sync.WaitGroup

	for c := range *clients {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			}

			_, session, err := dialer.Dial(ctx, *url, nil)
			if err != nil {
				log.Printf("client %d: dial error: %v", clientID, err)
				totalErrors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			// Create streams
			var streamWG sync.WaitGroup
			for s := range *streams {
				streamWG.Add(1)
				go func(streamID int) {
					defer streamWG.Done()

					stream, err := session.OpenStreamSync(ctx)
					if err != nil {
						totalErrors.Add(1)
						return
					}

					msg := make([]byte, *msgSize)
					for i := range msg {
						msg[i] = byte(i % 256)
					}

					// Length-prefix header
					header := make([]byte, 4)
					header[0] = byte(len(msg) >> 24)
					header[1] = byte(len(msg) >> 16)
					header[2] = byte(len(msg) >> 8)
					header[3] = byte(len(msg))

					deadline := time.Now().Add(*duration)
					for time.Now().Before(deadline) {
						sendStart := time.Now()

						// Write length-prefixed message
						if _, err := stream.Write(header); err != nil {
							totalErrors.Add(1)
							return
						}
						if _, err := stream.Write(msg); err != nil {
							totalErrors.Add(1)
							return
						}

						// Read response header
						respHeader := make([]byte, 4)
						n := 0
						for n < 4 {
							nn, err := stream.Read(respHeader[n:])
							if err != nil {
								totalErrors.Add(1)
								return
							}
							n += nn
						}

						respLen := int(respHeader[0])<<24 | int(respHeader[1])<<16 |
							int(respHeader[2])<<8 | int(respHeader[3])
						resp := make([]byte, respLen)
						n = 0
						for n < respLen {
							nn, err := stream.Read(resp[n:])
							if err != nil {
								totalErrors.Add(1)
								return
							}
							n += nn
						}

						latency := time.Since(sendStart)
						totalLatency.Add(int64(latency))
						totalMessages.Add(1)
						totalBytes.Add(int64(*msgSize))
					}

					stream.Close()
				}(s)
			}
			streamWG.Wait()
		}(c)
	}

	wg.Wait()
	elapsed := time.Since(start)

	msgs := totalMessages.Load()
	bytes := totalBytes.Load()
	errs := totalErrors.Load()
	avgLatency := time.Duration(0)
	if msgs > 0 {
		avgLatency = time.Duration(totalLatency.Load() / msgs)
	}

	fmt.Println()
	fmt.Println("=== Results ===")
	fmt.Printf("Duration:     %v\n", elapsed.Truncate(time.Millisecond))
	fmt.Printf("Clients:      %d\n", *clients)
	fmt.Printf("Streams:      %d total\n", *clients**streams)
	fmt.Printf("Messages:     %d\n", msgs)
	fmt.Printf("Throughput:   %.0f msg/s\n", float64(msgs)/elapsed.Seconds())
	fmt.Printf("Bandwidth:    %.2f MB/s\n", float64(bytes)/elapsed.Seconds()/1024/1024)
	fmt.Printf("Avg Latency:  %v\n", avgLatency.Truncate(time.Microsecond))
	fmt.Printf("Errors:       %d\n", errs)
}
