package auth

import (
	"context"
	"log"
	"net/http"
	"strings"

	"Communicate/internal/handler"
	"Communicate/internal/store/db/queries/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const (
	ctxUserID    ctxKey = "user_id"
	ctxSessionID ctxKey = "session_id"
)

// RequireAuth is middleware that validates the bearer token and injects
// user_id and session_id into the request context.
func RequireAuth(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				handler.WriteError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			s, err := sessions.GetByAccessToken(r.Context(), pool, token)
			if err != nil {
				log.Printf("auth middleware: %v", err)
				handler.WriteError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if s == nil {
				handler.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserID, s.UserID)
			ctx = context.WithValue(ctx, ctxSessionID, s.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

// SessionIDFromContext returns the current session ID from the context.
func SessionIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxSessionID).(string)
	return id
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}
