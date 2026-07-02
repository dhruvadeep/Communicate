package handler

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strings"

	"Communicate/internal/store/sqlite/queries/ratelimit"
)

// RateLimit wraps a handler with rate limiting on failed attempts.
// endpoint is a label like "login", "register", "forgot-password".
// On success (2xx/3xx), any lock is cleared.
// On failure (4xx/5xx), the attempt is recorded.
func RateLimit(database *sql.DB, endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		id := ip + ":" + endpoint

		status := ratelimit.Check(database, id, endpoint)
		if status.Locked {
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"error":       "too many attempts",
				"locked_until": status.LockedUntil.UTC().Format("2006-01-02T15:04:05Z"),
			})
			return
		}

		// Wrap the response writer to capture the status code.
		rw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		// On success, clear any lock. On failure, record the attempt.
		if rw.status >= 200 && rw.status < 400 {
			ratelimit.ClearLock(database, id, endpoint)
		} else {
			ratelimit.RecordAttempt(database, id, endpoint)
			// Add rate limit headers so the client knows.
			s := ratelimit.Check(database, id, endpoint)
			if s.AttemptsLeft > 0 {
				rw.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", s.AttemptsLeft))
			}
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip := strings.SplitN(fwd, ",", 2)[0]
		return strings.TrimSpace(ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
