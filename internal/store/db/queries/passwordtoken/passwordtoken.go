package passwordtoken

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Token represents a password_reset_tokens row.
type Token struct {
	ID        string
	UserID    string
	TokenHash string
	CreatedAt time.Time
	UsedAt    *time.Time
	ExpiresAt time.Time
}

// HashPlaintext returns SHA-256 hex of a plaintext token.
func HashPlaintext(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

// ConstantTimeCompare returns true if a and b are equal.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CreateToken inserts a new password reset token (hashed).
func CreateToken(ctx context.Context, pool *pgxpool.Pool, userID, tokenPlaintext string, expiresAt time.Time) error {
	hash := HashPlaintext(tokenPlaintext)
	_, err := pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expiresAt)
	if err != nil {
		log.Printf("create password reset token: %v", err)
		return err
	}
	return nil
}

// GetValidByUserID returns the most recent non-expired, unused token for a user.
func GetValidByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) (*Token, error) {
	t := &Token{}
	err := pool.QueryRow(ctx, `
		SELECT _id, user_id, token_hash, created_at, used_at, expires_at
		FROM password_reset_tokens
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.CreatedAt, &t.UsedAt, &t.ExpiresAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("get password reset token: %v", err)
		return nil, err
	}
	return t, nil
}

// MarkUsed sets used_at on a token so it can't be reused.
func MarkUsed(ctx context.Context, pool *pgxpool.Pool, tokenID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW() WHERE _id = $1
	`, tokenID)
	if err != nil {
		log.Printf("mark password reset token used: %v", err)
		return err
	}
	return nil
}

// GetByHash looks up a valid (non-expired, unused) token by its hash.
func GetByHash(ctx context.Context, pool *pgxpool.Pool, hash string) (*Token, error) {
	t := &Token{}
	err := pool.QueryRow(ctx, `
		SELECT _id, user_id, token_hash, created_at, used_at, expires_at
		FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
	`, hash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.CreatedAt, &t.UsedAt, &t.ExpiresAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("get password reset token by hash: %v", err)
		return nil, err
	}
	return t, nil
}

// InvalidateAllForUser marks all unused tokens as used for a user.
func InvalidateAllForUser(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW()
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	if err != nil {
		log.Printf("invalidate password reset tokens: %v", err)
		return err
	}
	return nil
}
