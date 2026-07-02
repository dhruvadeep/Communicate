package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	db   *sql.DB
	once sync.Once
)

// Open returns a singleton SQLite database for rate limiting.
// The database file is created in the project root as "communicate_rate.db".
func Open() (*sql.DB, error) {
	var err error
	once.Do(func() {
		dir, e := os.Getwd()
		if e != nil {
			err = fmt.Errorf("getwd: %w", e)
			return
		}
		path := filepath.Join(dir, "communicate_rate.db")

		db, err = sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			err = fmt.Errorf("open sqlite: %w", err)
			return
		}

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if e := db.Ping(); e != nil {
			err = fmt.Errorf("ping sqlite: %w", e)
			return
		}

		log.Println("sqlite connected for rate limiting")
	})
	return db, err
}

// DB returns the shared sql.DB handle (must call Open first).
func DB() *sql.DB { return db }

// Close shuts down the SQLite connection.
func Close() {
	if db != nil {
		db.Close()
	}
}
