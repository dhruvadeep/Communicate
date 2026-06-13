package tables

import "Communicate/internal/store/db/models/structure"

var Users = structure.Migration{
	TableName: "users",
	CreateTableCommand: `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		CREATE TABLE IF NOT EXISTS users (
			_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			username VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			email_verified_at TIMESTAMPTZ,
			profile_image_url TEXT,
			profile_image_object_key TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_login_at TIMESTAMPTZ,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			deleted_at TIMESTAMPTZ
		)
	`,
	DeleteTableCommand: `
		DROP TABLE IF EXISTS users
	`,
	DeleteRowsCommand: `
		TRUNCATE TABLE users RESTART IDENTITY
	`,
}
