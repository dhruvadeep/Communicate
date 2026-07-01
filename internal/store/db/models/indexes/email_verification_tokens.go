package indexes

import "Communicate/internal/store/db/models/structure"

var EmailVerificationTokens = structure.Index{
	TableName: "email_verification_tokens",
	CreateSQL: []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_email_verification_tokens_token ON email_verification_tokens(token)",
		"CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id ON email_verification_tokens(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_expires_at ON email_verification_tokens(expires_at)",
	},
	DropSQL: []string{
		"DROP INDEX IF EXISTS idx_email_verification_tokens_token",
		"DROP INDEX IF EXISTS idx_email_verification_tokens_user_id",
		"DROP INDEX IF EXISTS idx_email_verification_tokens_expires_at",
	},
}
