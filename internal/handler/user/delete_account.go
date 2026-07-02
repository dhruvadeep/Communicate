package user

import (
	"log"
	"net/http"

	"Communicate/internal/handler"
	"Communicate/internal/handler/auth"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DeleteAccount returns a handler for DELETE /users/me (soft delete).
func DeleteAccount(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if err := user.SoftDelete(r.Context(), pool, userID); err != nil {
			log.Printf("delete account: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Revoke all sessions.
		if err := sessions.RevokeAllForUser(r.Context(), pool, userID); err != nil {
			log.Printf("delete account: revoke sessions: %v", err)
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "account deleted",
		})
	}
}
