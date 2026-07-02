package sessions

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Session represents a user_sessions row.
type Session struct {
	ID                string
	UserID            string
	AccessTokenHash   string
	RefreshTokenHash  string
	RevokedAt         *time.Time
	CreatedAt         time.Time
	LastUsedAt        time.Time
	IPAddress         *string
	UserAgent         *string
}

// TokenPair holds the plaintext tokens returned to the client.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

// HashToken returns the SHA-256 hex digest of a plaintext token.
func HashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

// ConstantTimeCompare returns true if the two strings are equal (timing-safe).
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CreateSession inserts a new session row and returns the plaintext tokens.
// The caller is responsible for returning the plaintext to the client.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID, accessToken, refreshToken, ip, ua string) (*TokenPair, error) {
	accessHash := HashToken(accessToken)
	refreshHash := HashToken(refreshToken)

	var ipAddr, userAgent *string
	if ip != "" {
		ipAddr = &ip
	}
	if ua != "" {
		userAgent = &ua
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO user_sessions (user_id, access_token_hash, refresh_token_hash, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, accessHash, refreshHash, ipAddr, userAgent)
	if err != nil {
		log.Printf("create session: %v", err)
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339),
	}, nil
}

// GetByAccessToken looks up a non-revoked session by access token hash.
func GetByAccessToken(ctx context.Context, pool *pgxpool.Pool, accessToken string) (*Session, error) {
	hash := HashToken(accessToken)
	s := &Session{}
	err := pool.QueryRow(ctx, `
		SELECT _id, user_id, access_token_hash, refresh_token_hash,
		       revoked_at, created_at, last_used_at, ip_address, user_agent
		FROM user_sessions
		WHERE access_token_hash = $1 AND revoked_at IS NULL
	`, hash).Scan(
		&s.ID, &s.UserID, &s.AccessTokenHash, &s.RefreshTokenHash,
		&s.RevokedAt, &s.CreatedAt, &s.LastUsedAt, &s.IPAddress, &s.UserAgent,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("get session by access token: %v", err)
		return nil, err
	}
	return s, nil
}

// GetByRefreshToken looks up a non-revoked session by refresh token hash.
func GetByRefreshToken(ctx context.Context, pool *pgxpool.Pool, refreshToken string) (*Session, error) {
	hash := HashToken(refreshToken)
	s := &Session{}
	err := pool.QueryRow(ctx, `
		SELECT _id, user_id, access_token_hash, refresh_token_hash,
		       revoked_at, created_at, last_used_at, ip_address, user_agent
		FROM user_sessions
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL
	`, hash).Scan(
		&s.ID, &s.UserID, &s.AccessTokenHash, &s.RefreshTokenHash,
		&s.RevokedAt, &s.CreatedAt, &s.LastUsedAt, &s.IPAddress, &s.UserAgent,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		log.Printf("get session by refresh token: %v", err)
		return nil, err
	}
	return s, nil
}

// Touch updates last_used_at on a session.
func Touch(ctx context.Context, pool *pgxpool.Pool, sessionID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE user_sessions SET last_used_at = NOW() WHERE _id = $1
	`, sessionID)
	if err != nil {
		log.Printf("touch session: %v", err)
		return err
	}
	return nil
}

// RevokeSession marks a single session as revoked.
func RevokeSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = NOW() WHERE _id = $1
	`, sessionID)
	if err != nil {
		log.Printf("revoke session: %v", err)
		return err
	}
	return nil
}

// RevokeAllForUser revokes every session belonging to a user (logout everywhere).
func RevokeAllForUser(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	if err != nil {
		log.Printf("revoke all sessions for user: %v", err)
		return err
	}
	return nil
}
