# Rules

## Project structure

Each concern gets its own folder. Within each folder, one file per entity. Everything is grouped by what it _is_, not by what feature it belongs to.

```
internal/
├── config/          — env loading, singleton config
├── handler/         — HTTP handlers
│   ├── helpers.go   — shared response writers (WriteJSON, WriteError)
│   ├── logger.go    — request-logging middleware
│   ├── health.go    — GET /health
│   ├── auth/        — auth handlers (register, login, …)
│   └── user/        — user handlers (profile image, …)
├── mail/            — SMTP mailer
└── store/
    ├── db/
    │   ├── database.go          — connection pool (Open, Close)
    │   ├── migrate/
    │   │   ├── tables.go        — table migration runner
    │   │   └── indexes.go       — index creation/drop runner
    │   ├── models/
    │   │   ├── structure/
    │   │   │   └── model.go     — Migration & Index struct definitions
    │   │   ├── tables/          — one file per table, SQL for create/drop/truncate
    │   │   └── indexes/         — one file per table, index SQL
    │   └── queries/             — data-access functions
    │       └── user/
    │           └── user.go      — User struct + Create/Exists/…
    └── r2/          — Cloudflare R2 client
```

## Adding a new database table

1. Create a file `internal/store/db/models/tables/<thing>.go` exporting a `structure.Migration`.
2. Create a file `internal/store/db/models/indexes/<thing>.go` exporting a `structure.Index`.
3. Add the migration to `tables/all.go` and the index to `indexes/all.go`.
4. Both are picked up automatically by `cmd/server/main.go` at startup.

### Migration struct

```go
var Thing = structure.Migration{
    TableName:          "things",
    CreateTableCommand: `CREATE TABLE IF NOT EXISTS things (…)`,
    DeleteTableCommand: `DROP TABLE IF EXISTS things`,
    DeleteRowsCommand:  `TRUNCATE TABLE things RESTART IDENTITY`,
}
```

### Index struct

```go
var Thing = structure.Index{
    TableName: "things",
    CreateSQL: []string{"CREATE INDEX IF NOT EXISTS …"},
    DropSQL:   []string{"DROP INDEX IF EXISTS …"},
}
```

## Adding queries for an entity

Put them in `internal/store/db/queries/<entity>/<entity>.go`. The package name is the entity name. Keep the struct and all data-access functions for that entity in a single file.

## Adding an HTTP handler

- Place it under `internal/handler/<domain>/<operation>.go`.
- Use `handler.WriteJSON` / `handler.WriteError` for responses.
- Validate inputs early, return structured errors.
- Handlers receive dependencies (pool, clients) via closures.

## Logging

Every function that touches I/O (database, network, file system) must log errors. Do not silently return them — the caller may not log either.

In query functions:
```go
if err != nil {
    log.Printf("describe what failed: %v", err)
    return nil, err
}
```

In handlers, log before writing the error response:
```go
if err != nil {
    log.Printf("create user: %v", err)
    handler.WriteError(w, http.StatusInternalServerError, "internal server error")
    return
}
```

## General principles

- **Keep it simple and practical.** Don't over-engineer. One file per entity, one folder per concern.
- **No ORM.** Raw SQL via `pgx/v5`.
- **Stdlib-first.** Prefer the standard library; reach for dependencies only when they carry real weight.
- **Idempotent migrations.** Use `IF NOT EXISTS` / `IF EXISTS` so migrations can run repeatedly.
