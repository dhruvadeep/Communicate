package auth

import (
	"log"
	"net/http"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/emailtoken"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifyEmail returns an http.HandlerFunc that handles GET /auth/verify-email.
// Returns an HTML page since this is clicked from an email client.
func VerifyEmail(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			handler.WriteHTML(w, http.StatusBadRequest, "Missing Token", "The verification link is missing the token parameter.")
			return
		}

		t, err := emailtoken.GetValidToken(r.Context(), pool, token)
		if err != nil {
			log.Printf("get verification token: %v", err)
			handler.WriteHTML(w, http.StatusInternalServerError, "Server Error", "Something went wrong. Please try again later.")
			return
		}
		if t == nil {
			handler.WriteHTML(w, http.StatusBadRequest, "Invalid or Expired", "This verification link is no longer valid. Please request a new one.")
			return
		}

		if err := emailtoken.MarkEmailVerified(r.Context(), pool, t.UserID); err != nil {
			log.Printf("mark email verified: %v", err)
			handler.WriteHTML(w, http.StatusInternalServerError, "Server Error", "Something went wrong. Please try again later.")
			return
		}

		if err := emailtoken.ConsumeToken(r.Context(), pool, t.ID); err != nil {
			log.Printf("consume token: %v", err)
		}

		handler.WriteHTML(w, http.StatusOK, "Email Verified",
			"Your email address has been verified successfully. You can now close this page and log in.")
	}
}
