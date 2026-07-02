package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

// Refresh returns an http.HandlerFunc that handles POST /auth/refresh.
// It rotates both tokens: the old session is deleted, a new one is created.
func Refresh(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.RefreshToken = strings.TrimSpace(req.RefreshToken)
		if req.RefreshToken == "" {
			handler.WriteError(w, http.StatusBadRequest, "refresh_token is required")
			return
		}

		old, err := sessions.GetByRefreshToken(r.Context(), pool, req.RefreshToken)
		if err != nil {
			log.Printf("refresh: lookup: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if old == nil {
			handler.WriteError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}

		// Revoke the old session.
		if err := sessions.RevokeSession(r.Context(), pool, old.ID); err != nil {
			log.Printf("refresh: revoke old: %v", err)
		}

		// Create a new session with fresh tokens.
		newAccess := generateToken()
		newRefresh := generateToken()

		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		ua := r.Header.Get("User-Agent")

		pair, err := sessions.CreateSession(r.Context(), pool, old.UserID, newAccess, newRefresh, ip, ua)
		if err != nil {
			log.Printf("refresh: create session: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		handler.WriteJSON(w, http.StatusOK, refreshResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
			ExpiresAt:    time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339),
		})
	}
}
