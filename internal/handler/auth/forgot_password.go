package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/mail"
	"Communicate/internal/store/db/queries/passwordtoken"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
)

type forgotRequest struct {
	Email string `json:"email"`
}

// ForgotPassword returns an http.HandlerFunc that handles POST /auth/forgot-password.
func ForgotPassword(pool *pgxpool.Pool, mailer *mail.Mailer, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req forgotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || !strings.Contains(req.Email, "@") {
			// Don't reveal validation details — return the same response as "not found".
			handler.WriteJSON(w, http.StatusOK, map[string]string{
				"message": "if that email is registered, a reset link has been sent",
			})
			return
		}

		u, err := user.FindByEmail(r.Context(), pool, req.Email)
		if err != nil {
			log.Printf("forgot password: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil || !u.IsActive || u.DeletedAt != nil {
			// Don't leak whether the email exists.
			handler.WriteJSON(w, http.StatusOK, map[string]string{
				"message": "if that email is registered, a reset link has been sent",
			})
			return
		}

		// Invalidate any old unused reset tokens for this user.
		if err := passwordtoken.InvalidateAllForUser(r.Context(), pool, u.ID); err != nil {
			log.Printf("forgot password: invalidate old tokens: %v", err)
		}

		token := generateToken()
		expiresAt := time.Now().Add(1 * time.Hour)

		if err := passwordtoken.CreateToken(r.Context(), pool, u.ID, token, expiresAt); err != nil {
			log.Printf("forgot password: create token: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		go sendPasswordResetEmail(mailer, baseURL, u.Email, u.Username, token)

		handler.WriteJSON(w, http.StatusOK, map[string]string{
			"message": "if that email is registered, a reset link has been sent",
		})
	}
}

func sendPasswordResetEmail(mailer *mail.Mailer, baseURL, to, username, token string) {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, token)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f4;font-family:Arial,Helvetica,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f4;padding:30px 0;">
<tr><td align="center">
<table width="480" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:6px;overflow:hidden;">

  <tr>
    <td style="background-color:#1a1a2e;padding:28px 32px;text-align:center;">
      <span style="color:#ffffff;font-size:22px;font-weight:bold;">Communicate</span>
    </td>
  </tr>

  <tr>
    <td style="padding:32px 32px 20px;color:#333333;font-size:15px;line-height:1.6;">
      <p style="margin:0 0 12px;">Hi <strong>%s</strong>,</p>
      <p style="margin:0 0 20px;">A password reset was requested for your account. Click the button below to set a new password.</p>

      <table cellpadding="0" cellspacing="0" style="margin:0 auto;">
        <tr>
          <td align="center" style="background-color:#1a1a2e;border-radius:4px;">
            <a href="%s" target="_blank" style="display:inline-block;padding:13px 36px;color:#ffffff;font-size:15px;font-weight:bold;text-decoration:none;">Reset Password</a>
          </td>
        </tr>
      </table>

      <p style="margin:24px 0 0;font-size:13px;color:#777777;">
        Or copy and paste this link:<br>
        <a href="%s" style="color:#1a1a2e;word-break:break-all;">%s</a>
      </p>

      <p style="margin:20px 0 0;font-size:13px;color:#999999;">
        This link expires in 1 hour. If you did not request this, you can safely ignore this email.
      </p>
    </td>
  </tr>

  <tr>
    <td style="background-color:#fafafa;padding:16px 32px;text-align:center;font-size:12px;color:#aaaaaa;border-top:1px solid #eeeeee;">
      Communicate
    </td>
  </tr>

</table>
</td></tr>
</table>
</body>
</html>`, username, resetURL, resetURL, resetURL)

	textBody := fmt.Sprintf(
		"Hi %s,\n\n"+
			"A password reset was requested for your account. Click the link below to set a new password:\n\n"+
			"%s\n\n"+
			"This link expires in 1 hour.\n\n"+
			"If you did not request this, you can safely ignore this email.\n\n"+
			"Communicate\n",
		username, resetURL,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mailer.Send(ctx, "Reset your password", htmlBody, textBody, to); err != nil {
		log.Printf("send password reset email to %s: %v", to, err)
	}
}
