package sqlite

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations creates the rate_limits table if it doesn't exist.
func RunMigrations(database *sql.DB) error {
	const query = `
		CREATE TABLE IF NOT EXISTS rate_limits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			identifier TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			attempted_at INTEGER NOT NULL,
			locked_until INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_rate_limits_lookup
			ON rate_limits(identifier, endpoint, attempted_at);
	`

	if _, err := database.Exec(query); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}

	log.Println("sqlite migrations applied")
	return nil
}
