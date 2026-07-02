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

// ResetPassword returns an http.HandlerFunc that handles POST /auth/reset-password.
func ResetPassword(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Support both JSON body and query param for the token.
		var req resetRequest

		if r.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		}

		// Token can also come from query param (from email link).
		if req.Token == "" {
			req.Token = r.URL.Query().Get("token")
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

		// Find the user who has this token. Since tokens are hashed, we iterate
		// over valid tokens for the user we find. But we don't know the user yet.
		// Strategy: the token itself links to a specific user — we find the user
		// by looking at whose tokens match the hash of the incoming plaintext.
		tokenHash := passwordtoken.HashPlaintext(req.Token)

		// Find the token row by hash.
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

		// Hash the new password.
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("reset password: hash: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Update password.
		if err := user.UpdatePassword(r.Context(), pool, t.UserID, string(hash)); err != nil {
			log.Printf("reset password: update: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Mark token as used so it can't be reused.
		if err := passwordtoken.MarkUsed(r.Context(), pool, t.ID); err != nil {
			log.Printf("reset password: mark used: %v", err)
		}

		// Invalidate all other unused reset tokens for this user.
		if err := passwordtoken.InvalidateAllForUser(r.Context(), pool, t.UserID); err != nil {
			log.Printf("reset password: invalidate all: %v", err)
		}

		// Log out all sessions — security best practice after password change.
		if err := sessions.RevokeAllForUser(r.Context(), pool, t.UserID); err != nil {
			log.Printf("reset password: revoke sessions: %v", err)
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "password reset successfully — all sessions have been revoked, please log in again",
		})
	}
}