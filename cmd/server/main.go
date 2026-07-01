package main

import (
	"context"
	"log"

	"Communicate/internal/store/db"
	"Communicate/internal/store/db/migrate"
	"Communicate/internal/store/db/models/indexes"
	"Communicate/internal/store/db/models/tables"
)

func main() {
	ctx := context.Background()

	database, err := db.Open(ctx)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := migrate.RunMigrations(ctx, database.Pool(), tables.All...); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// After tables are created, run indexes:
	if err := migrate.RunIndexes(ctx, database.Pool(), indexes.All...); err != nil {
		log.Fatalf("failed to run indexes: %v", err)
	}
	//
	// To drop all indexes:
	// if err := migrate.DropIndexes(ctx, database.Pool(), indexes.All...); err != nil {
	// 	log.Fatalf("failed to drop indexes: %v", err)
	// }

	// To drop all tables:
	// if err := migrate.DropTables(ctx, database.Pool(), tables.All...); err != nil {
	// 	log.Fatalf("failed to drop all tables: %v", err)
	// }

	log.Println("database connected and migrations applied")

	// To add a new table:
	// 1. create another file in internal/store/db/models/tables
	// 2. define a Migration with TableName, CreateTableCommand, DeleteTableCommand, and DeleteRowsCommand
	// 3. add it to tables.All

	// To clear rows without dropping the table:
	// if err := migrate.ClearTables(ctx, database.Pool(), tables.Users); err != nil {
	// 	log.Fatalf("failed to clear users table: %v", err)
	// }
}
