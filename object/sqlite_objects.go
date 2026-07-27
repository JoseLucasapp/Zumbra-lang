package object

import "fmt"

// SQLiteExecResult is the portable result of an INSERT, UPDATE, DELETE or DDL statement.
type SQLiteExecResult struct {
	LastInsertID int64
	RowsAffected int64
}

// SQLiteDatabaseRuntime is implemented by the platform SQLite adapter.
type SQLiteDatabaseRuntime interface {
	Exec(query string, params []Object) (SQLiteExecResult, error)
	Query(query string, params []Object) ([]map[string]Object, error)
	Prepare(query string) (SQLiteStatementRuntime, error)
	Begin() (SQLiteTransactionRuntime, error)
	Close() error
	IsOpen() bool
}

type SQLiteStatementRuntime interface {
	Exec(params []Object) (SQLiteExecResult, error)
	Query(params []Object) ([]map[string]Object, error)
	Close() error
	IsOpen() bool
	SQL() string
}

type SQLiteTransactionRuntime interface {
	Exec(query string, params []Object) (SQLiteExecResult, error)
	Query(query string, params []Object) ([]map[string]Object, error)
	Prepare(query string) (SQLiteStatementRuntime, error)
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
