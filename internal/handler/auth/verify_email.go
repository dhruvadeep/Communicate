package auth

import (
	"log"
	"net/http"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/emailtoken"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifyEmail returns an http.HandlerFunc that handles GET /auth/verify-email.
func VerifyEmail(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			handler.WriteError(w, http.StatusBadRequest, "token is required")
			return
		}

		t, err := emailtoken.GetValidToken(r.Context(), pool, token)
		if err != nil {
			log.Printf("get verification token: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if t == nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}

		if err := emailtoken.MarkEmailVerified(r.Context(), pool, t.UserID); err != nil {
			log.Printf("mark email verified: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if err := emailtoken.ConsumeToken(r.Context(), pool, t.ID); err != nil {
			log.Printf("consume token: %v", err)
			// Email is already verified — non-fatal if token cleanup fails.
		}

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "email verified successfully",
		})
	}
}
