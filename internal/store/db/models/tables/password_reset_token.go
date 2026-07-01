package tables

import "Communicate/internal/store/db/models/structure"

var PasswordResetTokens = structure.Migration{
	TableName: "password_reset_tokens",
	CreateTableCommand: `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			token_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			used_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ NOT NULL
		)
	`,
	DeleteTableCommand: `
		DROP TABLE IF EXISTS password_reset_tokens
	`,
	DeleteRowsCommand: `
		TRUNCATE TABLE password_reset_tokens RESTART IDENTITY
	`,
}
