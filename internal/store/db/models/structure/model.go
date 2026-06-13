package structure

// Migration keeps all SQL commands for one table in one place.
type Migration struct {
	TableName          string
	CreateTableCommand string
	DeleteTableCommand string
	DeleteRowsCommand  string
}
