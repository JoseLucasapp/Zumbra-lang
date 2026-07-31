package object

import "fmt"

// SQLiteExecResult is the portable result of an INSERT, UPDATE, DELETE or DDL statement.
type SQLiteExecResult struct {
	LastInsertID int64
	RowsAffected int64
}

// SQLiteDatabaseRuntime is implemented by the platform SQLite adapter.
type SQLiteDatabaseRuntime interface {
	Exec(query string, params Object) (SQLiteExecResult, error)
	Query(query string, params Object) ([]map[string]Object, error)
	QueryStream(query string, params Object) (SQLRowsRuntime, error)
	Prepare(query string) (SQLiteStatementRuntime, error)
	Begin() (SQLiteTransactionRuntime, error)
	Migrate(migrations []SQLiteMigration) (int64, error)
	SchemaVersion() (int64, error)
	Backup(path string) error
	Restore(path string) error
	IntegrityCheck() (string, error)
	Close() error
	IsOpen() bool
}

type SQLiteMigration struct {
	Version int64
	Name    string
	SQL     string
}

type SQLiteStatementRuntime interface {
	Exec(params Object) (SQLiteExecResult, error)
	Query(params Object) ([]map[string]Object, error)
	QueryStream(params Object) (SQLRowsRuntime, error)
	ParameterCount() int
	ColumnNames() []string
	Close() error
	IsOpen() bool
	SQL() string
}

type SQLiteTransactionRuntime interface {
	Exec(query string, params Object) (SQLiteExecResult, error)
	Query(query string, params Object) ([]map[string]Object, error)
	QueryStream(query string, params Object) (SQLRowsRuntime, error)
	Prepare(query string) (SQLiteStatementRuntime, error)
	Savepoint(name string) error
	RollbackTo(name string) error
	Release(name string) error
	Commit() error
	Rollback() error
	Active() bool
}

type SQLiteDatabase struct {
	Runtime SQLiteDatabaseRuntime
	Path    string
}

func (d *SQLiteDatabase) Type() ObjectType { return SQLITE_DATABASE_OBJ }
func (d *SQLiteDatabase) Inspect() string {
	if d == nil || d.Runtime == nil {
		return "SQLiteDatabase<nil>"
	}
	return fmt.Sprintf("SQLiteDatabase<%s open=%t>", d.Path, d.Runtime.IsOpen())
}

type SQLiteStatement struct {
	Runtime SQLiteStatementRuntime
}

func (s *SQLiteStatement) Type() ObjectType { return SQLITE_STATEMENT_OBJ }
func (s *SQLiteStatement) Inspect() string {
	if s == nil || s.Runtime == nil {
		return "SQLiteStatement<nil>"
	}
	return fmt.Sprintf("SQLiteStatement<open=%t sql=%q>", s.Runtime.IsOpen(), s.Runtime.SQL())
}

type SQLiteTransaction struct {
	Runtime SQLiteTransactionRuntime
}

func (t *SQLiteTransaction) Type() ObjectType { return SQLITE_TRANSACTION_OBJ }
func (t *SQLiteTransaction) Inspect() string {
	if t == nil || t.Runtime == nil {
		return "SQLiteTransaction<nil>"
	}
	return fmt.Sprintf("SQLiteTransaction<active=%t>", t.Runtime.Active())
}
