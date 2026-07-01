package indexes

import "Communicate/internal/store/db/models/structure"

var PasswordResetTokens = structure.Index{
	TableName: "password_reset_tokens",
	CreateSQL: []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash)",
		"CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at)",
	},
	DropSQL: []string{
		"DROP INDEX IF EXISTS idx_password_reset_tokens_token_hash",
		"DROP INDEX IF EXISTS idx_password_reset_tokens_user_id",
		"DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at",
	},
}
