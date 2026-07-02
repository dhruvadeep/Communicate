package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Health returns an http.HandlerFunc that handles GET /health.
func Health(pool *pgxpool.Pool, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbStart := time.Now()
		var ok int
		if err := pool.QueryRow(ctx, "SELECT 1").Scan(&ok); err != nil {
			log.Printf("health db ping: %v", err)
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":    "unhealthy",
				"db":        "down",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		dbPing := time.Since(dbStart)

		WriteJSON(w, http.StatusOK, map[string]any{
			"status":         "healthy",
			"db":             "up",
			"db_ping_ms":     dbPing.Seconds() * 1000,
			"uptime":         formatUptime(time.Since(startTime)),
			"uptime_seconds": time.Since(startTime).Seconds(),
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func formatUptime(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	return strings.Join(parts, " ")
}
