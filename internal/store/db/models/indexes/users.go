package indexes

import "Communicate/internal/store/db/models/structure"

var Users = structure.Index{
	TableName: "users",
	CreateSQL: []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_email_verified_at ON users(email_verified_at)",
	},
	DropSQL: []string{
		"DROP INDEX IF EXISTS idx_users_email",
		"DROP INDEX IF EXISTS idx_users_username",
		"DROP INDEX IF EXISTS idx_users_email_verified_at",
	},
}
