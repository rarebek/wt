// Example: AI token streaming server.
//
// Demonstrates WebTransport's advantage for LLM streaming:
// - Multiple independent streams per session (one per agent/conversation)
// - No head-of-line blocking between streams
// - Datagrams for metadata (token counts, latency stats)
//
// This simulates an AI gateway that streams tokens from multiple
// concurrent LLM calls without them blocking each other.
package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

type StreamRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	MaxTok int    `json:"max_tokens"`
}

type TokenChunk struct {
	Token string `json:"token"`
	Index int    `json:"index"`
	Done  bool   `json:"done"`
}

type UsageStats struct {
	TotalTokens   int     `json:"total_tokens"`
	AvgLatencyMS  float64 `json:"avg_latency_ms"`
	ActiveStreams  int     `json:"active_streams"`
}

// Simulated vocabulary for fake token generation
var vocabulary = []string{
	"The ", "quick ", "brown ", "fox ", "jumps ", "over ", "the ", "lazy ", "dog. ",
	"In ", "a ", "world ", "where ", "AI ", "agents ", "collaborate, ",
	"efficiency ", "is ", "key ", "to ", "success. ",
	"WebTransport ", "enables ", "low-latency ", "streaming ",
	"of ", "tokens ", "across ", "multiple ", "independent ", "channels. ",
}

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	log.Printf("Certificate hash: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))

	server.Handle("/ai/stream", handleAIStream)

	log.Printf("AI streaming server listening on %s", server.Addr())
	log.Fatal(server.ListenAndServe())
}

func handleAIStream(c *wt.Context) {
	slog.Info("AI client connected", "id", c.ID())

	// Track stats and send via datagrams periodically
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stats := UsageStats{
					ActiveStreams: 0, // would track in real app
					AvgLatencyMS: rand.Float64() * 50,
				}
				data, _ := json.Marshal(stats)
				if err := c.SendDatagram(data); err != nil {
					return
				}
			case <-c.Context().Done():
				return
			}
		}
	}()

	// Each client stream = one completion request
	// Multiple streams can run concurrently without blocking each other
	for {
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		go handleCompletionStream(stream)
	}
}

func handleCompletionStream(s *wt.Stream) {
	defer s.Close()

	// Read the request
	reqData, err := s.ReadMessage()
	if err != nil {
		return
	}

	var req StreamRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		return
	}

	slog.Info("completion request",
		"model", req.Model,
		"prompt_len", len(req.Prompt),
		"max_tokens", req.MaxTok,
	)

	maxTokens := req.MaxTok
	if maxTokens <= 0 || maxTokens > 100 {
		maxTokens = 20
	}

	// Simulate token-by-token streaming
	for i := range maxTokens {
		// Simulate LLM inference latency (10-80ms per token)
		time.Sleep(time.Duration(10+rand.IntN(70)) * time.Millisecond)

		token := vocabulary[rand.IntN(len(vocabulary))]
		chunk := TokenChunk{
			Token: token,
			Index: i,
			Done:  i == maxTokens-1,
		}

		data, _ := json.Marshal(chunk)
		if err := s.WriteMessage(data); err != nil {
			return
		}
	}
}
