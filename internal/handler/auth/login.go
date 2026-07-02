package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    string       `json:"expires_at"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	Username        string  `json:"username"`
	ProfileImageURL *string `json:"profile_image_url"`
	EmailVerified   bool    `json:"email_verified"`
	CreatedAt       string  `json:"created_at"`
}

// Login returns an http.HandlerFunc that handles POST /auth/login.
func Login(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		if req.Email == "" || req.Password == "" {
			handler.WriteError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		u, err := user.FindByEmail(r.Context(), pool, req.Email)
		if err != nil {
			log.Printf("login: find user: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if u == nil {
			handler.WriteError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		if !u.IsActive || u.DeletedAt != nil {
			handler.WriteError(w, http.StatusUnauthorized, "account is disabled")
			return
		}

		if u.EmailVerifiedAt == nil {
			handler.WriteError(w, http.StatusForbidden, "email not verified — please check your inbox")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
			handler.WriteError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		accessToken := generateToken()
		refreshToken := generateToken()

		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		ua := r.Header.Get("User-Agent")

		pair, err := sessions.CreateSession(r.Context(), pool, u.ID, accessToken, refreshToken, ip, ua)
		if err != nil {
			log.Printf("login: create session: %v", err)
			handler.WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		handler.WriteJSON(w, http.StatusOK, loginResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
			ExpiresAt:    pair.ExpiresAt,
			User: userResponse{
				ID:              u.ID,
				Email:           u.Email,
				Username:        u.Username,
				ProfileImageURL: u.ProfileImageURL,
				EmailVerified:   true,
				CreatedAt:       u.CreatedAt.Format(time.RFC3339),
			},
		})
	}
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("generate token: %v (falling back to time-based)", err)
		// Fallback — should never happen, but don't crash.
		h := hex.EncodeToString([]byte(time.Now().String()))
		return h[:64]
	}
	return hex.EncodeToString(b)
}
