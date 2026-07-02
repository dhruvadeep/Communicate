package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/mail"
	"Communicate/internal/store/db/queries/emailtoken"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

type resendRequest struct {
	Email string `json:"email"`
}

// ResendVerification returns an http.HandlerFunc that handles POST /auth/resend-verification.
func ResendVerification(pool *pgxpool.Pool, mailer *mail.Mailer, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || !strings.Contains(req.Email, "@") {
			handler.WriteError(w, http.StatusBadRequest, "valid email is required")
			return
		}

		u, err := user.FindByEmail(r.Context(), pool, req.Email)
		if err != nil {
			log.Printf("resend verification: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Don't reveal whether the email exists.
		if u == nil {
			handler.WriteJSON(w, http.StatusOK, map[string]string{
				"message": "if that email is registered, a verification email has been sent",
			})
			return
		}

		// Already verified — no need to resend.
		if u.EmailVerifiedAt != nil {
			handler.WriteJSON(w, http.StatusOK, map[string]string{
				"message": "email is already verified",
			})
			return
		}

		// Remove any old tokens for this user so only the latest is valid.
		if err := emailtoken.DeleteTokensForUser(r.Context(), pool, u.ID); err != nil {
			log.Printf("resend verification: delete old tokens: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		token, err := emailtoken.Generate()
		if err != nil {
			log.Printf("resend verification: generate token: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		expiresAt := time.Now().Add(24 * time.Hour)
		if err := emailtoken.CreateToken(r.Context(), pool, u.ID, token, expiresAt); err != nil {
			log.Printf("resend verification: create token: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		go sendVerificationEmail(mailer, baseURL, u.Email, u.Username, token)

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "verification email sent",
		})
	}
}
