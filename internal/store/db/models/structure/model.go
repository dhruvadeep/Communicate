package structure

// Migration keeps all SQL commands for one table in one place.
type Migration struct {
	TableName          string
	CreateTableCommand string
	DeleteTableCommand string
	DeleteRowsCommand  string
}

// Index groups CREATE INDEX and DROP INDEX commands for a single table.
type Index struct {
	TableName string
	CreateSQL []string
	DropSQL   []string
}
