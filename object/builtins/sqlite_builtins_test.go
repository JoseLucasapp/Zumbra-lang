package builtins

import (
	"path/filepath"
	"testing"

	"zumbra/object"
)

func requireSQLiteValue(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if errObj, ok := value.(*object.Error); ok {
		t.Fatalf("SQLite builtin failed: %s", errObj.Message)
	}
	return value
}

func sqliteTestArray(values ...object.Object) *object.Array { return &object.Array{Elements: values} }

func sqliteTestDictValue(t *testing.T, dict *object.Dict, key string) object.Object {
	t.Helper()
	pair, ok := dict.Pairs[(&object.String{Value: key}).DictKey()]
	if !ok {
		t.Fatalf("missing key %q in %s", key, dict.Inspect())
	}
	return pair.Value
}

func TestZ12SQLiteMemoryParametersRowsStatementsAndTransactions(t *testing.T) {
	database := requireSQLiteValue(t, SQLiteMemoryBuiltin().Fn()).(*object.SQLiteDatabase)
	defer SQLiteCloseBuiltin().Fn(database)

	requireSQLiteValue(t, SQLiteExecBuiltin().Fn(database, NewString(`create table users (
        id integer primary key,
        name text not null,
        score real,
        active integer,
        payload blob
    )`), sqliteTestArray()))

	statement := requireSQLiteValue(t, SQLitePrepareBuiltin().Fn(database, NewString("insert into users(name, score, active, payload) values (?, ?, ?, ?)"))).(*object.SQLiteStatement)
	payload := &object.ByteArray{Data: []byte{0x4e, 0x45, 0x53, 0x1a}}
	inserted := requireSQLiteValue(t, SQLiteStatementExecBuiltin().Fn(statement, sqliteTestArray(NewString("Lucas"), NewFloat(9.5), NewBoolean(true), payload))).(*object.Dict)
	if got := sqliteTestDictValue(t, inserted, "rowsAffected").(*object.Integer).Value; got != 1 {
		t.Fatalf("rowsAffected=%d", got)
	}
	requireSQLiteValue(t, SQLiteStatementExecBuiltin().Fn(statement, sqliteTestArray(NewString("Empty"), NewFloat(0), NewBoolean(false), &object.ByteArray{Data: []byte{}})))
	requireSQLiteValue(t, SQLiteStatementCloseBuiltin().Fn(statement))

	rows := requireSQLiteValue(t, SQLiteQueryBuiltin().Fn(database, NewString("select name, score, active, payload from users where name = ?"), sqliteTestArray(NewString("Lucas")))).(*object.Array)
	if len(rows.Elements) != 1 {
		t.Fatalf("expected one row, got %d", len(rows.Elements))
	}
	row := rows.Elements[0].(*object.Dict)
	if got := sqliteTestDictValue(t, row, "name").(*object.String).Value; got != "Lucas" {
		t.Fatalf("name=%q", got)
	}
	if got := sqliteTestDictValue(t, row, "score").(*object.Float).Value; got != 9.5 {
		t.Fatalf("score=%v", got)
	}
	if got := sqliteTestDictValue(t, row, "active").(*object.Integer).Value; got != 1 {
		t.Fatalf("active=%d", got)
	}
	if got := sqliteTestDictValue(t, row, "payload").(*object.ByteArray).Data; string(got) != "NES\x1a" {
		t.Fatalf("payload=%v", got)
	}
	emptyRows := requireSQLiteValue(t, SQLiteQueryBuiltin().Fn(database, NewString("select payload from users where name = ?"), sqliteTestArray(NewString("Empty")))).(*object.Array)
	emptyPayload := sqliteTestDictValue(t, emptyRows.Elements[0].(*object.Dict), "payload").(*object.ByteArray).Data
	if len(emptyPayload) != 0 {
		t.Fatalf("empty payload length=%d", len(emptyPayload))
	}

	transaction := requireSQLiteValue(t, SQLiteBeginBuiltin().Fn(database)).(*object.SQLiteTransaction)
	requireSQLiteValue(t, SQLiteTransactionExecBuiltin().Fn(transaction, NewString("insert into users(name, score, active) values (?, ?, ?)"), sqliteTestArray(NewString("Rollback"), NewInteger(1), NewBoolean(false))))
	requireSQLiteValue(t, SQLiteRollbackBuiltin().Fn(transaction))
	count := requireSQLiteValue(t, SQLiteQueryBuiltin().Fn(database, NewString("select count(*) as total from users"), sqliteTestArray())).(*object.Array)
	total := sqliteTestDictValue(t, count.Elements[0].(*object.Dict), "total").(*object.Integer).Value
	if total != 2 {
		t.Fatalf("rollback did not preserve row count: %d", total)
	}
}

func TestZ12SQLiteFileDatabaseAndSafeParameterCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zumbra.sqlite")
	database := requireSQLiteValue(t, SQLiteOpenBuiltin().Fn(NewString(path))).(*object.SQLiteDatabase)
	requireSQLiteValue(t, SQLiteExecBuiltin().Fn(database, NewString("create table values_table(value text)"), sqliteTestArray()))
	failed := SQLiteExecBuiltin().Fn(database, NewString("insert into values_table(value) values (?)"), sqliteTestArray())
	if _, ok := failed.(*object.Error); !ok {
		t.Fatalf("expected parameter-count error, got %T", failed)
	}
	requireSQLiteValue(t, SQLiteCloseBuiltin().Fn(database))
	if SQLiteIsOpenBuiltin().Fn(database).(*object.Boolean).Value {
		t.Fatal("database remained open")
	}
}
