package user

import (
	"log"
	"net/http"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/handler/auth"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

type meResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	Username        string  `json:"username"`
	ProfileImageURL *string `json:"profile_image_url"`
	EmailVerified   bool    `json:"email_verified"`
	CreatedAt       string  `json:"created_at"`
}

// Me returns an http.HandlerFunc that handles GET /users/me.
func Me(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		u, err := user.FindByID(r.Context(), pool, userID)
		if err != nil {
			log.Printf("me: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil {
			handler.WriteError(w, http.StatusNotFound, "user not found")
			return
		}

		emailVerified := u.EmailVerifiedAt != nil

		handler.WriteJSON(w, http.StatusOK, meResponse{
			ID:              u.ID,
			Email:           u.Email,
			Username:        u.Username,
			ProfileImageURL: u.ProfileImageURL,
			EmailVerified:   emailVerified,
			CreatedAt:       u.CreatedAt.Format(time.RFC3339),
		})
	}
}
