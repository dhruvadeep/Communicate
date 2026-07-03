package handler

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// WriteHTML writes a simple styled HTML page. Used for browser-facing endpoints
// like email verification and password reset (clicked from email links).
func WriteHTML(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + title + `</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:Arial,Helvetica,sans-serif;background:#f4f4f4;display:flex;align-items:center;justify-content:center;min-height:100vh}
.card{background:#fff;border-radius:8px;padding:40px 32px;text-align:center;max-width:420px;width:100%;box-shadow:0 2px 8px rgba(0,0,0,0.08)}
.card h1{font-size:20px;color:#1a1a2e;margin-bottom:12px}
.card p{font-size:15px;color:#555;line-height:1.6}
.status{display:inline-block;width:48px;height:48px;border-radius:50%;margin-bottom:16px;line-height:48px;font-size:24px}
.ok{background:#e8f5e9;color:#2e7d32}
.err{background:#ffebee;color:#c62828}
</style>
</head>
<body>
<div class="card">
<div class="status ` + statusClass(status) + `">` + statusIcon(status) + `</div>
<h1>` + title + `</h1>
<p>` + message + `</p>
</div>
</body>
</html>`))
}

func statusClass(status int) string {
	if status >= 200 && status < 400 {
		return "ok"
	}
	return "err"
}

func statusIcon(status int) string {
	if status >= 200 && status < 400 {
		return "&#10003;"
	}
	return "&#10007;"
}
