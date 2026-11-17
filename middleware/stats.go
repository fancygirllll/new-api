package middleware

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// HTTPStats stores HTTP statistics.
type HTTPStats struct {
	activeConnections int64
}

var globalStats = &HTTPStats{}

// StatsMiddleware is the statistics middleware.
func StatsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment the active connection count.
		atomic.AddInt64(&globalStats.activeConnections, 1)

		// Ensure the connection count is decremented when the request finishes.
		defer func() {
			atomic.AddInt64(&globalStats.activeConnections, -1)
		}()

		c.Next()
	}
}

// StatsInfo is the statistics structure.
type StatsInfo struct {
	ActiveConnections int64 `json:"active_connections"`
}

// GetStats returns the current statistics.
func GetStats() StatsInfo {
	return StatsInfo{
		ActiveConnections: atomic.LoadInt64(&globalStats.activeConnections),
	}
}
