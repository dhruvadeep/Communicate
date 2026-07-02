package auth

import (
	"log"
	"net/http"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Logout returns an http.HandlerFunc that handles POST /auth/logout.
// It revokes ALL sessions for the authenticated user (logout everywhere).
func Logout(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if err := sessions.RevokeAllForUser(r.Context(), pool, userID); err != nil {
			log.Printf("logout: revoke all: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "logged out from all devices",
		})
	}
}
