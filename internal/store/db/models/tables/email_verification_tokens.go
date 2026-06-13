package tables

import (
	"Communicate/internal/store/db/models/structure"
)

var EmailVerificationTokens = structure.Migration{
	TableName: "email_verification_tokens",
	CreateTableCommand: `
		CREATE TABLE IF NOT EXISTS email_verification_tokens (
			_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			token VARCHAR(255) NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`,
	DeleteTableCommand: `
		DROP TABLE IF EXISTS email_verification_tokens
	`,
	DeleteRowsCommand: `
		TRUNCATE TABLE email_verification_tokens RESTART IDENTITY
	`,
}
