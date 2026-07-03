package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/passwordtoken"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword handles both GET and POST /auth/reset-password.
// GET (from email link): shows an HTML form to enter a new password.
// POST (form submit or API call): validates the token and updates the password.
func ResetPassword(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.URL.Query().Get("token"))

		// GET — show the reset form (clicked from email).
		if r.Method == http.MethodGet {
			if token == "" {
				handler.WriteHTML(w, http.StatusBadRequest, "Missing Token", "The reset link is missing the token parameter.")
				return
			}
			writeResetForm(w, token)
			return
		}

		// POST — process the reset.
		var req resetRequest
		if r.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		} else {
			// Form submission from the HTML page.
			req.Token = r.FormValue("token")
			req.Password = r.FormValue("password")
		}

		if req.Token == "" {
			req.Token = token
		}
		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			handler.WriteError(w, http.StatusBadRequest, "token is required")
			return
		}
		if len(req.Password) < 8 {
			handler.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}

		tokenHash := passwordtoken.HashPlaintext(req.Token)

		t, err := passwordtoken.GetByHash(r.Context(), pool, tokenHash)
		if err != nil {
			log.Printf("reset password: lookup token: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if t == nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("reset password: hash: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if err := user.UpdatePassword(r.Context(), pool, t.UserID, string(hash)); err != nil {
			log.Printf("reset password: update: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if err := passwordtoken.MarkUsed(r.Context(), pool, t.ID); err != nil {
			log.Printf("reset password: mark used: %v", err)
		}

		if err := passwordtoken.InvalidateAllForUser(r.Context(), pool, t.UserID); err != nil {
			log.Printf("reset password: invalidate all: %v", err)
		}

		if err := sessions.RevokeAllForUser(r.Context(), pool, t.UserID); err != nil {
			log.Printf("reset password: revoke sessions: %v", err)
		}

		// If it's a browser form post, show HTML. Otherwise JSON.
		if r.Header.Get("Content-Type") == "application/json" {
			handler.WriteJSON(w, http.StatusOK, map[string]string{
				"message": "password reset successfully — all sessions have been revoked, please log in again",
			})
		} else {
			handler.WriteHTML(w, http.StatusOK, "Password Reset",
				"Your password has been reset successfully. All sessions have been revoked. You can now close this page and log in.")
		}
	}
}

func writeResetForm(w http.ResponseWriter, token string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Reset Password</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:Arial,Helvetica,sans-serif;background:#f4f4f4;display:flex;align-items:center;justify-content:center;min-height:100vh}
.card{background:#fff;border-radius:8px;padding:40px 32px;max-width:400px;width:100%;box-shadow:0 2px 8px rgba(0,0,0,0.08)}
.card h1{font-size:20px;color:#1a1a2e;margin-bottom:20px;text-align:center}
input{width:100%;padding:10px 12px;margin-bottom:12px;border:1px solid #ddd;border-radius:4px;font-size:14px}
button{width:100%;padding:12px;border:none;border-radius:4px;background:#1a1a2e;color:#fff;font-size:15px;font-weight:bold;cursor:pointer}
. hint{font-size:12px;color:#999;text-align:center;margin-top:12px}
</style>
</head>
<body>
<div class="card">
<h1>Reset Password</h1>
<form method="POST">
<input type="hidden" name="token" value="` + token + `">
<input type="password" name="password" placeholder="New password (min 8 characters)" minlength="8" required autofocus>
<button type="submit">Set New Password</button>
</form>
<p class="hint">Enter a new password for your account.</p>
</div>
</body>
</html>`))
}
