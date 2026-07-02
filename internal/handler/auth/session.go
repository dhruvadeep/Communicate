package auth

import (
	"log"
	"net/http"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Session returns an http.HandlerFunc that handles GET /auth/session.
// The client sends its access token; the server says whether it's still valid
// and returns the current user info (so the frontend can hydrate without relogin).
func Session(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		u, err := user.FindByID(r.Context(), pool, userID)
		if err != nil {
			log.Printf("session: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil || !u.IsActive || u.DeletedAt != nil {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Touch the session so last_used_at stays fresh.
		sessionID := SessionIDFromContext(r.Context())
		if sessionID != "" {
			if err := sessions.Touch(r.Context(), pool, sessionID); err != nil {
				log.Printf("session: touch: %v", err)
			}
		}

		emailVerified := u.EmailVerifiedAt != nil

		handler.WriteJSON(w, http.StatusOK, userResponse{
			ID:              u.ID,
			Email:           u.Email,
			Username:        u.Username,
			ProfileImageURL: u.ProfileImageURL,
			EmailVerified:   emailVerified,
			CreatedAt:       u.CreatedAt.Format(time.RFC3339),
		})
	}
}
