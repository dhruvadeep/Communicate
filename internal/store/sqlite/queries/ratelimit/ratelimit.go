package ratelimit

import (
	"database/sql"
	"log"
	"time"
)

// Status tells the caller whether to allow, block, or if the client is locked.
type Status struct {
	Allowed      bool
	Locked       bool
	LockedUntil  time.Time
	AttemptsLeft int
}

var maxAttempts int
var window time.Duration
var lockDuration time.Duration

// Configure sets the rate limit parameters (called once at startup).
func Configure(max int, w, lock time.Duration) {
	maxAttempts = max
	window = w
	lockDuration = lock
}

// Check returns the current rate limit status for an identifier+endpoint pair.
// It does NOT record an attempt — call RecordAttempt after a failure.
func Check(database *sql.DB, identifier, endpoint string) Status {
	now := time.Now().Unix()
	cutoff := time.Now().Add(-window).Unix()

	// Check if currently locked.
	var lockedUntil int64
	database.QueryRow(`
		SELECT locked_until FROM rate_limits
		WHERE identifier = ? AND endpoint = ? AND locked_until > ?
		ORDER BY locked_until DESC LIMIT 1
	`, identifier, endpoint, now).Scan(&lockedUntil)

	if lockedUntil > 0 {
		return Status{
			Allowed:     false,
			Locked:      true,
			LockedUntil: time.Unix(lockedUntil, 0),
		}
	}

	// Count recent failed attempts.
	var count int
	database.QueryRow(`
		SELECT COUNT(*) FROM rate_limits
		WHERE identifier = ? AND endpoint = ? AND attempted_at > ?
	`, identifier, endpoint, cutoff).Scan(&count)

	attemptsLeft := maxAttempts - count
	if attemptsLeft < 0 {
		attemptsLeft = 0
	}

	return Status{
		Allowed:      true,
		AttemptsLeft: attemptsLeft,
	}
}

// RecordAttempt logs a failed attempt. If this pushes the count over the max,
// the identifier+endpoint is locked for lockDuration.
func RecordAttempt(database *sql.DB, identifier, endpoint string) {
	now := time.Now().Unix()

	if _, err := database.Exec(`
		INSERT INTO rate_limits (identifier, endpoint, attempted_at)
		VALUES (?, ?, ?)
	`, identifier, endpoint, now); err != nil {
		log.Printf("ratelimit: record attempt: %v", err)
		return
	}

	// Check if we need to lock.
	cutoff := time.Now().Add(-window).Unix()
	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM rate_limits
		WHERE identifier = ? AND endpoint = ? AND attempted_at > ? AND locked_until = 0
	`, identifier, endpoint, cutoff).Scan(&count); err != nil {
		log.Printf("ratelimit: count after insert: %v", err)
		return
	}

	if count >= maxAttempts {
		lockUntil := time.Now().Add(lockDuration).Unix()
		if _, err := database.Exec(`
			UPDATE rate_limits SET locked_until = ?
			WHERE identifier = ? AND endpoint = ? AND attempted_at > ? AND locked_until = 0
		`, lockUntil, identifier, endpoint, cutoff); err != nil {
			log.Printf("ratelimit: lock: %v", err)
		}
	}
}

// ClearLock removes a lock early (e.g. after successful login).
func ClearLock(database *sql.DB, identifier, endpoint string) {
	if _, err := database.Exec(`
		UPDATE rate_limits SET locked_until = 0
		WHERE identifier = ? AND endpoint = ?
	`, identifier, endpoint); err != nil {
		log.Printf("ratelimit: clear lock: %v", err)
	}
}

// Cleanup deletes entries older than 24 hours. Call periodically.
func Cleanup(database *sql.DB) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	if _, err := database.Exec(`DELETE FROM rate_limits WHERE attempted_at < ?`, cutoff); err != nil {
		log.Printf("ratelimit: cleanup: %v", err)
	}
}
