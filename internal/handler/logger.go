package handler

import (
	"log"
	"net/http"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// Logger wraps an http.Handler and logs every request with method, path,
// status code (color-coded), and elapsed time.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		elapsed := time.Since(start)

		log.Printf("%s | %s | %s%3d%s | %s%10v%s | %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			statusColor(wrapped.status), wrapped.status, colorReset,
			colorCyan, elapsed, colorReset,
			r.URL.Path,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func statusColor(code int) string {
	switch {
	case code >= 500:
		return colorRed
	case code >= 400:
		return colorYellow
	default:
		return colorGreen
	}
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
