package middleware

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger is a Gin middleware that emits a structured JSON log line per request.
func Logger() gin.HandlerFunc {
	log := json.NewEncoder(os.Stdout)
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		latency := time.Since(start)
		entry := map[string]interface{}{
			"level":   "info",
			"ts":      time.Now().UTC().Format(time.RFC3339),
			"method":  c.Request.Method,
			"path":    path,
			"status":  c.Writer.Status(),
			"latency": latency.String(),
			"ip":      c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			entry["error"] = c.Errors.Last().Err.Error()
		}
		_ = log.Encode(entry)
		c.Header("X-Response-Time", latency.String())
	}
}
