package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/mail"
	"Communicate/internal/store/db/queries/emailtoken"
	"Communicate/internal/store/db/queries/user"
	"Communicate/internal/verify"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Email           string  `json:"email"`
	Username        string  `json:"username"`
	Password        string  `json:"password"`
	ProfileImageURL *string `json:"profile_image_url"`
}

type registerResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	Username        string  `json:"username"`
	ProfileImageURL *string `json:"profile_image_url"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// Register returns an http.HandlerFunc that handles POST /auth/register.
func Register(pool *pgxpool.Pool, verifier *verify.Verifier, mailer *mail.Mailer, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		req.Username = strings.TrimSpace(req.Username)

		if req.Email == "" || !strings.Contains(req.Email, "@") {
			handler.WriteError(w, http.StatusBadRequest, "valid email is required")
			return
		}
		if len(req.Username) < 3 || len(req.Username) > 50 {
			handler.WriteError(w, http.StatusBadRequest, "username must be between 3 and 50 characters")
			return
		}
		if len(req.Password) < 8 {
			handler.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}

		// Verify the email is real and deliverable before we touch the database.
		vresult := verifier.Verify(req.Email)
		if !vresult.Valid {
			handler.WriteError(w, http.StatusBadRequest, vresult.Reason)
			return
		}

		exists, err := user.ExistsByEmail(r.Context(), pool, req.Email)
		if err != nil {
			log.Printf("check email exists: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if exists {
			handler.WriteError(w, http.StatusConflict, "email already taken")
			return
		}

		exists, err = user.ExistsByUsername(r.Context(), pool, req.Username)
		if err != nil {
			log.Printf("check username exists: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if exists {
			handler.WriteError(w, http.StatusConflict, "username already taken")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("hash password: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		u, err := user.CreateUser(r.Context(), pool, req.Email, req.Username, string(hash), req.ProfileImageURL)
		if err != nil {
			if isUniqueViolation(err) {
				handler.WriteError(w, http.StatusConflict, "email or username already taken")
				return
			}
			log.Printf("create user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// Generate and store a verification token, then email it.
		token, err := emailtoken.Generate()
		if err != nil {
			log.Printf("generate verification token: %v", err)
			// User is created but token generation failed — don't block the response.
			// The user can request a new verification email later.
		} else {
			expiresAt := time.Now().Add(24 * time.Hour)
			if err := emailtoken.CreateToken(r.Context(), pool, u.ID, token, expiresAt); err != nil {
				log.Printf("store verification token: %v", err)
			} else {
				go sendVerificationEmail(mailer, baseURL, u.Email, u.Username, token)
			}
		}

		handler.WriteJSON(w, http.StatusCreated, map[string]any{
			"user": registerResponse{
				ID:              u.ID,
				Email:           u.Email,
				Username:        u.Username,
				ProfileImageURL: u.ProfileImageURL,
				CreatedAt:       u.CreatedAt.Format(time.RFC3339),
				UpdatedAt:       u.UpdatedAt.Format(time.RFC3339),
			},
		})
	}
}

func sendVerificationEmail(mailer *mail.Mailer, baseURL, to, username, token string) {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", baseURL, token)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f4;font-family:Arial,Helvetica,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f4;padding:30px 0;">
<tr><td align="center">
<table width="480" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:6px;overflow:hidden;">

  <!-- Header -->
  <tr>
    <td style="background-color:#1a1a2e;padding:28px 32px;text-align:center;">
      <span style="color:#ffffff;font-size:22px;font-weight:bold;">Communicate</span>
    </td>
  </tr>

  <!-- Body -->
  <tr>
    <td style="padding:32px 32px 20px;color:#333333;font-size:15px;line-height:1.6;">
      <p style="margin:0 0 12px;">Hi <strong>%s</strong>,</p>
      <p style="margin:0 0 20px;">Thanks for signing up. Please verify your email address by clicking the button below.</p>

      <!-- Button -->
      <table cellpadding="0" cellspacing="0" style="margin:0 auto;">
        <tr>
          <td align="center" style="background-color:#1a1a2e;border-radius:4px;">
            <a href="%s" target="_blank" style="display:inline-block;padding:13px 36px;color:#ffffff;font-size:15px;font-weight:bold;text-decoration:none;">Verify Email Address</a>
          </td>
        </tr>
      </table>

      <p style="margin:24px 0 0;font-size:13px;color:#777777;">
        Or copy and paste this link into your browser:<br>
        <a href="%s" style="color:#1a1a2e;word-break:break-all;">%s</a>
      </p>

      <p style="margin:20px 0 0;font-size:13px;color:#999999;">
        This link expires in 24 hours. If you did not create this account, you can safely ignore this email.
      </p>
    </td>
  </tr>

  <!-- Footer -->
  <tr>
    <td style="background-color:#fafafa;padding:16px 32px;text-align:center;font-size:12px;color:#aaaaaa;border-top:1px solid #eeeeee;">
      Communicate
    </td>
  </tr>

</table>
</td></tr>
</table>
</body>
</html>`, username, verifyURL, verifyURL, verifyURL)

	textBody := fmt.Sprintf(
		"Hi %s,\n\n"+
			"Thanks for signing up. Please verify your email address by clicking the link below:\n\n"+
			"%s\n\n"+
			"This link expires in 24 hours.\n\n"+
			"If you did not create this account, you can safely ignore this email.\n\n"+
			"Communicate\n",
		username, verifyURL,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mailer.Send(ctx, "Verify your email address", htmlBody, textBody, to); err != nil {
		log.Printf("send verification email to %s: %v", to, err)
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
