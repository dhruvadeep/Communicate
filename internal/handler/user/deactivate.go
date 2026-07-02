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

// Deactivate returns a handler for POST /users/me/deactivate (toggles is_active).
func Deactivate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		u, err := user.FindByID(r.Context(), pool, userID)
		if err != nil {
			log.Printf("deactivate: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil {
			handler.WriteError(w, http.StatusNotFound, "user not found")
			return
		}

		newState := !u.IsActive
		if err := user.SetActive(r.Context(), pool, userID, newState); err != nil {
			log.Printf("deactivate: set active: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// If deactivating, revoke all sessions.
		if !newState {
			if err := sessions.RevokeAllForUser(r.Context(), pool, userID); err != nil {
				log.Printf("deactivate: revoke sessions: %v", err)
			}
		}

		msg := "account activated"
		if !newState {
			msg = "account deactivated — all sessions revoked"
		}

		handler.WriteJSON(w, http.StatusOK, map[string]any{
			"message":   msg,
			"is_active": newState,
		})
	}
}
