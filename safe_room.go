package wt

import "log/slog"

// SafeBroadcast sends a datagram to all room members, recovering from panics.
// Logs and skips any member that causes a panic (e.g., closed connection).
func (r *Room) SafeBroadcast(data []byte, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	r.ForEach(func(c *Context) {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Warn("broadcast panic recovered",
						"session", c.ID(),
						"room", r.Name(),
						"panic", rec,
					)
				}
			}()
			_ = c.SendDatagram(data)
		}()
	})
}

// SafeBroadcastExcept is SafeBroadcast but excludes the given session.
func (r *Room) SafeBroadcastExcept(data []byte, excludeID string, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	r.ForEach(func(c *Context) {
		if c.ID() == excludeID {
			return
		}
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Warn("broadcast panic recovered",
						"session", c.ID(),
						"room", r.Name(),
						"panic", rec,
					)
				}
			}()
			_ = c.SendDatagram(data)
		}()
	})
}
