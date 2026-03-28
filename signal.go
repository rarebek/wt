package wt

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ListenAndServeWithGracefulShutdown starts the server and handles SIGTERM/SIGINT
// for graceful shutdown. Active sessions are drained within the given timeout.
//
// Usage:
//
//	server := wt.New(...)
//	server.Handle("/app", handler)
//	wt.ListenAndServeWithGracefulShutdown(server, 30*time.Second)
func ListenAndServeWithGracefulShutdown(s *Server, drainTimeout time.Duration) error {
	logger := slog.Default()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ListenAndServe()
	}()

	// Wait for signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, starting graceful shutdown",
			"signal", sig.String(),
			"drain_timeout", drainTimeout.String(),
			"active_sessions", s.SessionCount(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
			return err
		}

		logger.Info("graceful shutdown complete",
			"remaining_sessions", s.SessionCount(),
		)
		return nil

	case err := <-errCh:
		return err
	}
}
