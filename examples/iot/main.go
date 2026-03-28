// Example: IoT telemetry gateway.
//
// Demonstrates WebTransport for IoT:
// - Self-signed certs (no CA needed for LAN devices)
// - Datagrams for sensor readings (fire-and-forget, battery-efficient)
// - Streams for commands/config (reliable, ordered)
// - Per-device session tracking
package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

type SensorReading struct {
	DeviceID    string  `json:"device_id"`
	Temperature float64 `json:"temp"`
	Humidity    float64 `json:"humidity"`
	Battery     int     `json:"battery_pct"`
	Timestamp   int64   `json:"ts"`
}

type DeviceCommand struct {
	Type    string `json:"type"` // "reboot", "update_interval", "calibrate"
	Payload string `json:"payload"`
}

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
		wt.WithIdleTimeout(5*time.Minute),
	)

	log.Printf("IoT Gateway starting")
	log.Printf("Certificate hash for devices: %s", server.CertHash())

	metrics := middleware.NewMetrics()

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Use(metrics.Middleware())
	server.Use(middleware.MaxSessions(10000, nil))

	// Device telemetry endpoint
	server.Handle("/device/{id}/telemetry", func(c *wt.Context) {
		deviceID := c.Param("id")
		c.Set("device_id", deviceID)

		slog.Info("device connected",
			"device", deviceID,
			"remote", c.RemoteAddr().String(),
		)

		// Receive sensor datagrams (unreliable — perfect for high-frequency readings)
		go func() {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}

				var reading SensorReading
				if err := json.Unmarshal(data, &reading); err != nil {
					continue
				}
				reading.DeviceID = deviceID

				// In a real app: write to time-series DB, check thresholds, etc.
				slog.Debug("sensor reading",
					"device", deviceID,
					"temp", reading.Temperature,
					"humidity", reading.Humidity,
					"battery", reading.Battery,
				)

				// Alert on low battery
				if reading.Battery < 10 {
					slog.Warn("low battery",
						"device", deviceID,
						"battery", reading.Battery,
					)
				}
			}
		}()

		// Accept command streams from management system
		// (server pushes commands to device via streams)
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				slog.Info("device disconnected", "device", deviceID)
				return
			}

			go func() {
				defer stream.Close()
				msg, err := stream.ReadMessage()
				if err != nil {
					return
				}

				var cmd DeviceCommand
				if err := json.Unmarshal(msg, &cmd); err != nil {
					return
				}

				slog.Info("device command received",
					"device", deviceID,
					"command", cmd.Type,
				)

				// Acknowledge
				ack, _ := json.Marshal(map[string]string{
					"status": "ok",
					"cmd":    cmd.Type,
				})
				_ = stream.WriteMessage(ack)
			}()
		}
	})

	// Management dashboard endpoint
	server.Handle("/dashboard", func(c *wt.Context) {
		slog.Info("dashboard client connected", "remote", c.RemoteAddr())

		// Push metrics every second
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				snap := metrics.Snapshot()
				data, _ := json.Marshal(map[string]any{
					"active_devices":  snap.ActiveSessions - 1, // subtract dashboard itself
					"total_connected": snap.TotalSessions,
				})
				if err := c.SendDatagram(data); err != nil {
					return
				}
			case <-c.Context().Done():
				return
			}
		}
	})

	log.Fatal(server.ListenAndServe())
}
