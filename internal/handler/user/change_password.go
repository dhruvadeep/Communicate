package user

import (
	"encoding/json"
	"log"
	"net/http"

	"Communicate/internal/handler"
	"Communicate/internal/handler/auth"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword returns a handler for POST /users/me/password.
func ChangePassword(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			handler.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req changePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if len(req.NewPassword) < 8 {
			handler.WriteError(w, http.StatusBadRequest, "new password must be at least 8 characters")
			return
		}

		u, err := user.FindByID(r.Context(), pool, userID)
		if err != nil {
			log.Printf("change password: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil {
			handler.WriteError(w, http.StatusNotFound, "user not found")
			return
		}

		// Verify current password.
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			handler.WriteError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("change password: hash: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if err := user.UpdatePassword(r.Context(), pool, userID, string(hash)); err != nil {
			log.Printf("change password: update: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Revoke all other sessions so old tokens can't be used.
		if err := sessions.RevokeAllForUser(r.Context(), pool, userID); err != nil {
			log.Printf("change password: revoke sessions: %v", err)
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "password changed — all sessions revoked, please log in again",
		})
	}
}
