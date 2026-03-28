package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/rarebek/wt"
)

// SessionTimeoutWithWarning returns middleware that closes sessions after
// the given duration, but sends a warning datagram before closing.
// This gives clients a chance to save state or reconnect.
func SessionTimeoutWithWarning(timeout, warningBefore time.Duration, logger *slog.Logger) wt.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *wt.Context, next wt.HandlerFunc) {
		warnAt := timeout - warningBefore

		// Warning timer
		warnTimer := time.AfterFunc(warnAt, func() {
			logger.Info("session timeout warning",
				"session", c.ID(),
				"closing_in", warningBefore.String(),
			)
			// Send warning datagram
			_ = c.SendDatagram([]byte(`{"type":"timeout_warning","seconds_remaining":` +
				fmt.Sprintf("%d", int(warningBefore.Seconds())) + `}`))
		})

		// Close timer
		closeTimer := time.AfterFunc(timeout, func() {
			logger.Info("session timeout",
				"session", c.ID(),
				"after", timeout.String(),
			)
			_ = c.CloseWithError(408, "session timeout")
		})

		defer func() {
			warnTimer.Stop()
			closeTimer.Stop()
		}()

		next(c)
	}
}
