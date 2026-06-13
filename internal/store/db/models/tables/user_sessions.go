package tables

import (
	"Communicate/internal/store/db/models/structure"
)

var UserSessions = structure.Migration{
	TableName: "user_sessions",
	CreateTableCommand: `
		CREATE TABLE IF NOT EXISTS user_sessions (
			_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			access_token_hash VARCHAR(255) NOT NULL,
			refresh_token_hash VARCHAR(255) NOT NULL,
			revoked_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ip_address TEXT,
			user_agent TEXT
		)
	`,
	DeleteTableCommand: `
		DROP TABLE IF EXISTS user_sessions
	`,
	DeleteRowsCommand: `
		TRUNCATE TABLE user_sessions RESTART IDENTITY
	`,
}
