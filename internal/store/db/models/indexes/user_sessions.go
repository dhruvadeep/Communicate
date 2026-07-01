package indexes

import "Communicate/internal/store/db/models/structure"

var UserSessions = structure.Index{
	TableName: "user_sessions",
	CreateSQL: []string{
		"CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_access_token_hash ON user_sessions(access_token_hash)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_refresh_token_hash ON user_sessions(refresh_token_hash)",
		"CREATE INDEX IF NOT EXISTS idx_user_sessions_revoked_at ON user_sessions(revoked_at)",
	},
	DropSQL: []string{
		"DROP INDEX IF EXISTS idx_user_sessions_user_id",
		"DROP INDEX IF EXISTS idx_user_sessions_access_token_hash",
		"DROP INDEX IF EXISTS idx_user_sessions_refresh_token_hash",
		"DROP INDEX IF EXISTS idx_user_sessions_revoked_at",
	},
}
