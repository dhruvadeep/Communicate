package emailtoken

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Token represents an email verification token row.
type Token struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Generate creates a cryptographically random hex token.
func Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateToken inserts a new verification token for a user.
func CreateToken(ctx context.Context, pool *pgxpool.Pool, userID, token string, expiresAt time.Time) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, userID, token, expiresAt)
	if err != nil {
		log.Printf("create email verification token: %v", err)
		return err
	}
	return nil
}

// GetValidToken looks up a non-expired token by its value.
// Returns nil if the token doesn't exist or has expired.
func GetValidToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Token, error) {
	t := &Token{}
	err := pool.QueryRow(ctx, `
		SELECT _id, user_id, token, expires_at, created_at
		FROM email_verification_tokens
		WHERE token = $1 AND expires_at > NOW()
	`, token).Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("get email verification token: %v", err)
		return nil, err
	}
	return t, nil
}

// ConsumeToken deletes the token after successful verification.
func ConsumeToken(ctx context.Context, pool *pgxpool.Pool, tokenID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM email_verification_tokens WHERE _id = $1
	`, tokenID)
	if err != nil {
		log.Printf("consume email verification token: %v", err)
		return err
	}
	return nil
}

// DeleteTokensForUser removes all existing tokens for a user (used before resending).
func DeleteTokensForUser(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM email_verification_tokens WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Printf("delete tokens for user: %v", err)
		return err
	}
	return nil
}

// MarkEmailVerified sets email_verified_at on the user record.
func MarkEmailVerified(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE users SET email_verified_at = NOW(), updated_at = NOW()
		WHERE _id = $1
	`, userID)
	if err != nil {
		log.Printf("mark email verified: %v", err)
		return err
	}
	return nil
}
