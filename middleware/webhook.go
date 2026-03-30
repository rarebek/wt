package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rarebek/wt"
)

// WebhookEvent is sent to an external URL on session lifecycle events.
type WebhookEvent struct {
	Event     string `json:"event"` // "connect", "disconnect"
	SessionID string `json:"session_id"`
	RemoteAddr string `json:"remote_addr"`
	Path      string `json:"path"`
	Timestamp int64  `json:"timestamp"`
}

// Webhook returns middleware that POSTs session events to an external URL.
// Useful for analytics, logging pipelines, or triggering external workflows.
//
// Usage:
//
//	server.Use(middleware.Webhook("https://hooks.example.com/wt-events"))
func Webhook(url string, logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	client := &http.Client{Timeout: 5 * time.Second}

	return func(c *wt.Context, next wt.HandlerFunc) {
		// Fire connect event (async, don't block)
		go sendWebhook(client, url, WebhookEvent{
			Event:      "connect",
			SessionID:  c.ID(),
			RemoteAddr: c.RemoteAddr().String(),
			Path:       c.Request().URL.Path,
			Timestamp:  time.Now().UnixMilli(),
		}, logger)

		next(c)

		// Fire disconnect event
		go sendWebhook(client, url, WebhookEvent{
			Event:      "disconnect",
			SessionID:  c.ID(),
			RemoteAddr: c.RemoteAddr().String(),
			Path:       c.Request().URL.Path,
			Timestamp:  time.Now().UnixMilli(),
		}, logger)
	}
}

func sendWebhook(client *http.Client, url string, event WebhookEvent, logger *slog.Logger) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Debug("webhook failed", "url", url, "error", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Debug("webhook returned error", "url", url, "status", resp.StatusCode)
	}
}
