package models

import (
	"context"
	"fmt"

	"Communicate/internal/store/db/models/structure"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsTable = "schema_migrations"

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrations ...structure.Migration) error {
	if err := createMigrationTracker(ctx, pool); err != nil {
		return err
	}

	for _, migration := range migrations {
		alreadyCreated, err := migrationExists(ctx, pool, migration.TableName)
		if err != nil {
			return err
		}
		if alreadyCreated {
			continue
		}

		if _, err := pool.Exec(ctx, migration.CreateTableCommand); err != nil {
			return fmt.Errorf("create table for %s: %w", migration.TableName, err)
		}

		if err := saveMigrationRecord(ctx, pool, migration.TableName); err != nil {
			return err
		}
	}

	return nil
}

func DropTables(ctx context.Context, pool *pgxpool.Pool, migrations ...structure.Migration) error {
	if err := createMigrationTracker(ctx, pool); err != nil {
		return err
	}

	for i := len(migrations) - 1; i >= 0; i-- {
		migration := migrations[i]

		alreadyCreated, err := migrationExists(ctx, pool, migration.TableName)
		if err != nil {
			return err
		}
		if !alreadyCreated {
			continue
		}

		if _, err := pool.Exec(ctx, migration.DeleteTableCommand); err != nil {
			return fmt.Errorf("drop table for %s: %w", migration.TableName, err)
		}

		if err := deleteMigrationRecord(ctx, pool, migration.TableName); err != nil {
			return err
		}
	}

	return nil
}

func ClearTables(ctx context.Context, pool *pgxpool.Pool, migrations ...structure.Migration) error {
	for _, migration := range migrations {
		if migration.DeleteRowsCommand == "" {
			continue
		}

		if _, err := pool.Exec(ctx, migration.DeleteRowsCommand); err != nil {
			return fmt.Errorf("clear table for %s: %w", migration.TableName, err)
		}
	}

	return nil
}

func createMigrationTracker(ctx context.Context, pool *pgxpool.Pool) error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	return nil
}

func migrationExists(ctx context.Context, pool *pgxpool.Pool, tableName string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`

	var exists bool
	if err := pool.QueryRow(ctx, query, tableName).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s: %w", tableName, err)
	}

	return exists, nil
}

func saveMigrationRecord(ctx context.Context, pool *pgxpool.Pool, tableName string) error {
	query := fmt.Sprintf(`INSERT INTO %s (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, migrationsTable)
	if _, err := pool.Exec(ctx, query, tableName); err != nil {
		return fmt.Errorf("save migration record for %s: %w", tableName, err)
	}

	return nil
}

func deleteMigrationRecord(ctx context.Context, pool *pgxpool.Pool, tableName string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE name = $1`, migrationsTable)
	if _, err := pool.Exec(ctx, query, tableName); err != nil {
		return fmt.Errorf("delete migration record for %s: %w", tableName, err)
	}

	return nil
}
