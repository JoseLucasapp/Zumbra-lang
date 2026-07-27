package builtins

/*
#cgo linux LDFLAGS: -lsqlite3
#cgo darwin LDFLAGS: -lsqlite3
#include <stdlib.h>
#include <sqlite3.h>

static int zumbra_sqlite_bind_text(sqlite3_stmt *statement, int index, const char *value, int length) {
    return sqlite3_bind_text(statement, index, value, length, SQLITE_TRANSIENT);
}

static int zumbra_sqlite_bind_blob(sqlite3_stmt *statement, int index, const void *value, int length) {
    return sqlite3_bind_blob(statement, index, value, length, SQLITE_TRANSIENT);
}
*/
import "C"

import (
	"fmt"
	"math"
	"sync"
	"time"
	"unsafe"

	"zumbra/binarydata"
	"zumbra/object"
)

const sqliteMemoryPath = ":memory:"

type sqliteHandle struct {
	mu       sync.Mutex
	activeTx bool
	db       *C.sqlite3
	path     string
	open     bool
}

type sqliteStatementHandle struct {
	owner  *sqliteHandle
	tx     *sqliteTransactionHandle
	stmt   *C.sqlite3_stmt
	query  string
	closed bool
}

type sqliteTransactionHandle struct {
	owner  *sqliteHandle
	active bool
}

func sqliteMessage(db *C.sqlite3) string {
	if db == nil {
		return "SQLite database is unavailable"
	}
	return C.GoString(C.sqlite3_errmsg(db))
}

func openSQLite(path string) (*sqliteHandle, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var database *C.sqlite3
	flags := C.int(C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_CREATE | C.SQLITE_OPEN_FULLMUTEX | C.SQLITE_OPEN_URI)
	if code := C.sqlite3_open_v2(cPath, &database, flags, nil); code != C.SQLITE_OK {
		message := sqliteMessage(database)
		if database != nil {
			C.sqlite3_close_v2(database)
		}
		return nil, fmt.Errorf("open SQLite database %q: %s", path, message)
	}
	C.sqlite3_extended_result_codes(database, 1)
	C.sqlite3_busy_timeout(database, 5000)
	return &sqliteHandle{db: database, path: path, open: true}, nil
}

func (h *sqliteHandle) IsOpen() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.open && h.db != nil
}

func (h *sqliteHandle) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.open || h.db == nil {
		return nil
	}
	if h.activeTx {
		return fmt.Errorf("cannot close SQLite database with an active transaction")
	}
	if code := C.sqlite3_close_v2(h.db); code != C.SQLITE_OK {
		return fmt.Errorf("close SQLite database: %s", sqliteMessage(h.db))
	}
	h.db = nil
	h.open = false
	return nil
}

func (h *sqliteHandle) prepareLocked(query string) (*C.sqlite3_stmt, error) {
	if !h.open || h.db == nil {
		return nil, fmt.Errorf("SQLite database is closed")
	}
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	var statement *C.sqlite3_stmt
	if code := C.sqlite3_prepare_v2(h.db, cQuery, -1, &statement, nil); code != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare SQLite statement: %s", sqliteMessage(h.db))
	}
	if statement == nil {
		return nil, fmt.Errorf("SQLite query is empty")
	}
	return statement, nil
}

func (h *sqliteHandle) Exec(query string, params []object.Object) (object.SQLiteExecResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeTx {
		return object.SQLiteExecResult{}, fmt.Errorf("SQLite database has an active transaction; use the transaction handle")
	}
	statement, err := h.prepareLocked(query)
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	defer C.sqlite3_finalize(statement)
	return h.execStatementLocked(statement, params)
}

func (h *sqliteHandle) Query(query string, params []object.Object) ([]map[string]object.Object, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeTx {
		return nil, fmt.Errorf("SQLite database has an active transaction; use the transaction handle")
	}
	statement, err := h.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	defer C.sqlite3_finalize(statement)
	return h.queryStatementLocked(statement, params)
}

func (h *sqliteHandle) Prepare(query string) (object.SQLiteStatementRuntime, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeTx {
		return nil, fmt.Errorf("SQLite database has an active transaction; prepare through the transaction handle")
	}
	statement, err := h.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	return &sqliteStatementHandle{owner: h, stmt: statement, query: query}, nil
}

func (h *sqliteHandle) Begin() (object.SQLiteTransactionRuntime, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.open || h.db == nil {
		return nil, fmt.Errorf("SQLite database is closed")
	}
	if h.activeTx {
		return nil, fmt.Errorf("SQLite database already has an active transaction")
	}
	statement, err := h.prepareLocked("BEGIN IMMEDIATE")
	if err == nil {
		_, err = h.execStatementLocked(statement, nil)
		C.sqlite3_finalize(statement)
	}
	if err != nil {
		return nil, err
	}
	h.activeTx = true
	return &sqliteTransactionHandle{owner: h, active: true}, nil
}

func (h *sqliteHandle) execStatementLocked(statement *C.sqlite3_stmt, params []object.Object) (object.SQLiteExecResult, error) {
	if err := bindSQLiteParams(statement, params); err != nil {
		return object.SQLiteExecResult{}, err
	}
	code := C.sqlite3_step(statement)
	if code == C.SQLITE_ROW {
		return object.SQLiteExecResult{}, fmt.Errorf("SQLite statement returned rows; use query instead of exec")
	}
	if code != C.SQLITE_DONE {
		return object.SQLiteExecResult{}, fmt.Errorf("execute SQLite statement: %s", sqliteMessage(h.db))
	}
	return object.SQLiteExecResult{
		LastInsertID: int64(C.sqlite3_last_insert_rowid(h.db)),
		RowsAffected: int64(C.sqlite3_changes64(h.db)),
	}, nil
}

func (h *sqliteHandle) queryStatementLocked(statement *C.sqlite3_stmt, params []object.Object) ([]map[string]object.Object, error) {
	if err := bindSQLiteParams(statement, params); err != nil {
		return nil, err
	}
	columns := int(C.sqlite3_column_count(statement))
	rows := make([]map[string]object.Object, 0)
	for {
		code := C.sqlite3_step(statement)
		if code == C.SQLITE_DONE {
			return rows, nil
		}
		if code != C.SQLITE_ROW {
			return nil, fmt.Errorf("query SQLite statement: %s", sqliteMessage(h.db))
		}
		row := make(map[string]object.Object, columns)
		for column := 0; column < columns; column++ {
			name := C.GoString(C.sqlite3_column_name(statement, C.int(column)))
			row[name] = sqliteColumnObject(statement, column)
		}
		rows = append(rows, row)
	}
}

func bindSQLiteParams(statement *C.sqlite3_stmt, params []object.Object) error {
	C.sqlite3_reset(statement)
	C.sqlite3_clear_bindings(statement)
	expected := int(C.sqlite3_bind_parameter_count(statement))
	if len(params) != expected {
		return fmt.Errorf("SQLite statement expects %d parameters, got %d", expected, len(params))
	}
	for index, value := range params {
		if err := bindSQLiteValue(statement, index+1, value); err != nil {
			return fmt.Errorf("bind SQLite parameter %d: %w", index+1, err)
		}
	}
	return nil
}

func bindSQLiteValue(statement *C.sqlite3_stmt, index int, value object.Object) error {
	var code C.int
	switch current := value.(type) {
	case nil, *object.Null:
		code = C.sqlite3_bind_null(statement, C.int(index))
	case *object.Boolean:
		integer := C.sqlite3_int64(0)
		if current.Value {
			integer = 1
		}
		code = C.sqlite3_bind_int64(statement, C.int(index), integer)
	case *object.Integer:
		code = C.sqlite3_bind_int64(statement, C.int(index), C.sqlite3_int64(current.Value))
	case *object.FixedInteger:
		if current.Kind.Signed() {
			code = C.sqlite3_bind_int64(statement, C.int(index), C.sqlite3_int64(current.SignedValue()))
		} else if current.UnsignedValue() <= math.MaxInt64 {
			code = C.sqlite3_bind_int64(statement, C.int(index), C.sqlite3_int64(current.UnsignedValue()))
		} else {
			return fmt.Errorf("unsigned value %d exceeds SQLite signed INTEGER range", current.UnsignedValue())
		}
	case *object.Float:
		code = C.sqlite3_bind_double(statement, C.int(index), C.double(current.Value))
	case *object.String:
		text := C.CString(current.Value)
		defer C.free(unsafe.Pointer(text))
		code = C.zumbra_sqlite_bind_text(statement, C.int(index), text, C.int(len(current.Value)))
	case *object.Date:
		textValue := current.FullDate.Format(time.RFC3339Nano)
		if current.FullDate.IsZero() {
			textValue = fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", current.Year, current.Month, current.Day, current.Hour, current.Minute, current.Second)
		}
		text := C.CString(textValue)
		defer C.free(unsafe.Pointer(text))
		code = C.zumbra_sqlite_bind_text(statement, C.int(index), text, C.int(len(textValue)))
	default:
		data, err := binarydata.Bytes(value)
		if err != nil {
			return fmt.Errorf("unsupported Zumbra value %s", value.Type())
		}
		if len(data) == 0 {
			code = C.sqlite3_bind_zeroblob(statement, C.int(index), 0)
		} else {
			code = C.zumbra_sqlite_bind_blob(statement, C.int(index), unsafe.Pointer(&data[0]), C.int(len(data)))
		}
	}
	if code != C.SQLITE_OK {
		return fmt.Errorf("SQLite bind failed with code %d", int(code))
	}
	return nil
}

func sqliteColumnObject(statement *C.sqlite3_stmt, column int) object.Object {
	index := C.int(column)
	switch C.sqlite3_column_type(statement, index) {
	case C.SQLITE_INTEGER:
		return &object.Integer{Value: int64(C.sqlite3_column_int64(statement, index))}
	case C.SQLITE_FLOAT:
		return &object.Float{Value: float64(C.sqlite3_column_double(statement, index))}
	case C.SQLITE_TEXT:
		pointer := C.sqlite3_column_text(statement, index)
		length := C.sqlite3_column_bytes(statement, index)
		return &object.String{Value: C.GoStringN((*C.char)(unsafe.Pointer(pointer)), length)}
	case C.SQLITE_BLOB:
		pointer := C.sqlite3_column_blob(statement, index)
		length := C.sqlite3_column_bytes(statement, index)
		if length == 0 {
			return &object.ByteArray{Data: []byte{}}
		}
		return &object.ByteArray{Data: C.GoBytes(pointer, length)}
	default:
		return &object.Null{}
	}
}

func (s *sqliteStatementHandle) lock() (func(), error) {
	if s.owner == nil {
		return nil, fmt.Errorf("SQLite statement has no database")
	}
	s.owner.mu.Lock()
	if s.tx != nil {
		if !s.tx.active {
			s.owner.mu.Unlock()
			return nil, fmt.Errorf("SQLite transaction is no longer active")
		}
	} else if s.owner.activeTx {
		s.owner.mu.Unlock()
		return nil, fmt.Errorf("SQLite database has an active transaction; use a transaction statement")
	}
	return s.owner.mu.Unlock, nil
}

func (s *sqliteStatementHandle) Exec(params []object.Object) (object.SQLiteExecResult, error) {
	unlock, err := s.lock()
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	defer unlock()
	if s.closed || s.stmt == nil {
		return object.SQLiteExecResult{}, fmt.Errorf("SQLite statement is closed")
	}
	return s.owner.execStatementLocked(s.stmt, params)
}

func (s *sqliteStatementHandle) Query(params []object.Object) ([]map[string]object.Object, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if s.closed || s.stmt == nil {
		return nil, fmt.Errorf("SQLite statement is closed")
	}
	return s.owner.queryStatementLocked(s.stmt, params)
}

func (s *sqliteStatementHandle) Close() error {
	if s.owner == nil {
		return fmt.Errorf("SQLite statement has no database")
	}
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	if s.closed || s.stmt == nil {
		return nil
	}
	if code := C.sqlite3_finalize(s.stmt); code != C.SQLITE_OK {
		return fmt.Errorf("close SQLite statement: %s", sqliteMessage(s.owner.db))
	}
	s.stmt = nil
	s.closed = true
	return nil
}
func (s *sqliteStatementHandle) IsOpen() bool {
	if s == nil || s.owner == nil {
		return false
	}
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	return !s.closed && s.stmt != nil && s.owner.open && s.owner.db != nil
}
func (s *sqliteStatementHandle) SQL() string { return s.query }

func (t *sqliteTransactionHandle) Active() bool {
	if t == nil || t.owner == nil {
		return false
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	return t.active
}
func (t *sqliteTransactionHandle) Exec(query string, params []object.Object) (object.SQLiteExecResult, error) {
	if t == nil || t.owner == nil {
		return object.SQLiteExecResult{}, fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return object.SQLiteExecResult{}, fmt.Errorf("SQLite transaction is no longer active")
	}
	statement, err := t.owner.prepareLocked(query)
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	defer C.sqlite3_finalize(statement)
	return t.owner.execStatementLocked(statement, params)
}

func (t *sqliteTransactionHandle) Query(query string, params []object.Object) ([]map[string]object.Object, error) {
	if t == nil || t.owner == nil {
		return nil, fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return nil, fmt.Errorf("SQLite transaction is no longer active")
	}
	statement, err := t.owner.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	defer C.sqlite3_finalize(statement)
	return t.owner.queryStatementLocked(statement, params)
}

func (t *sqliteTransactionHandle) Prepare(query string) (object.SQLiteStatementRuntime, error) {
	if t == nil || t.owner == nil {
		return nil, fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return nil, fmt.Errorf("SQLite transaction is no longer active")
	}
	statement, err := t.owner.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	return &sqliteStatementHandle{owner: t.owner, tx: t, stmt: statement, query: query}, nil
}

func (t *sqliteTransactionHandle) finish(command string) error {
	if t == nil || t.owner == nil {
		return fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return fmt.Errorf("SQLite transaction is no longer active")
	}
	statement, err := t.owner.prepareLocked(command)
	if err == nil {
		_, err = t.owner.execStatementLocked(statement, nil)
		C.sqlite3_finalize(statement)
	}
	if err == nil {
		t.active = false
		t.owner.activeTx = false
	}
	return err
}

func (t *sqliteTransactionHandle) Commit() error   { return t.finish("COMMIT") }
func (t *sqliteTransactionHandle) Rollback() error { return t.finish("ROLLBACK") }

func sqliteParams(value object.Object, name string) ([]object.Object, *object.Error) {
	if value == nil || value.Type() == object.NULL_OBJ {
		return nil, nil
	}
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("%s parameters must be an array, got %s", name, value.Type())
	}
	return array.Elements, nil
}

func sqliteRowsObject(rows []map[string]object.Object) object.Object {
	elements := make([]object.Object, 0, len(rows))
	for _, row := range rows {
		pairs := make(map[object.DictKey]object.DictPair, len(row))
		for name, value := range row {
			key := &object.String{Value: name}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: value}
		}
		elements = append(elements, &object.Dict{Pairs: pairs})
	}
	return &object.Array{Elements: elements}
}

func sqliteExecResultObject(result object.SQLiteExecResult) object.Object {
	pairs := map[object.DictKey]object.DictPair{}
	for name, value := range map[string]int64{"lastInsertId": result.LastInsertID, "rowsAffected": result.RowsAffected} {
		key := &object.String{Value: name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Integer{Value: value}}
	}
	return &object.Dict{Pairs: pairs}
}

func SQLiteOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteOpen expects 1 argument, got %d", len(args))
		}
		path, ok := args[0].(*object.String)
		if !ok {
			return NewError("sqliteOpen path must be STRING, got %s", args[0].Type())
		}
		runtime, err := openSQLite(path.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteDatabase{Runtime: runtime, Path: path.Value}
	}}
}

func SQLiteMemoryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("sqliteMemory expects 0 arguments, got %d", len(args))
		}
		runtime, err := openSQLite(sqliteMemoryPath)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteDatabase{Runtime: runtime, Path: sqliteMemoryPath}
	}}
}

func sqliteDatabaseArg(value object.Object, name string) (*object.SQLiteDatabase, *object.Error) {
	database, ok := value.(*object.SQLiteDatabase)
	if !ok || database.Runtime == nil {
		return nil, NewError("%s expects SQLiteDatabase, got %s", name, value.Type())
	}
	return database, nil
}
func sqliteStatementArg(value object.Object, name string) (*object.SQLiteStatement, *object.Error) {
	statement, ok := value.(*object.SQLiteStatement)
	if !ok || statement.Runtime == nil {
		return nil, NewError("%s expects SQLiteStatement, got %s", name, value.Type())
	}
	return statement, nil
}
func sqliteTransactionArg(value object.Object, name string) (*object.SQLiteTransaction, *object.Error) {
	transaction, ok := value.(*object.SQLiteTransaction)
	if !ok || transaction.Runtime == nil {
		return nil, NewError("%s expects SQLiteTransaction, got %s", name, value.Type())
	}
	return transaction, nil
}
func sqliteQueryArgs(args []object.Object, name string) (string, []object.Object, *object.Error) {
	if len(args) < 2 || len(args) > 3 {
		return "", nil, NewError("%s expects 2 or 3 arguments, got %d", name, len(args))
	}
	query, ok := args[1].(*object.String)
	if !ok {
		return "", nil, NewError("%s query must be STRING, got %s", name, args[1].Type())
	}
	if len(args) == 2 {
		return query.Value, nil, nil
	}
	params, errObj := sqliteParams(args[2], name)
	return query.Value, params, errObj
}

func SQLiteExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, errObj := sqliteQueryArgs(args, "sqliteExec")
		if errObj != nil {
			return errObj
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqliteExec")
		if errObj != nil {
			return errObj
		}
		result, err := database.Runtime.Exec(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func SQLiteQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, errObj := sqliteQueryArgs(args, "sqliteQuery")
		if errObj != nil {
			return errObj
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqliteQuery")
		if errObj != nil {
			return errObj
		}
		rows, err := database.Runtime.Query(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLitePrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sqlitePrepare expects 2 arguments, got %d", len(args))
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqlitePrepare")
		if errObj != nil {
			return errObj
		}
		query, ok := args[1].(*object.String)
		if !ok {
			return NewError("sqlitePrepare query must be STRING, got %s", args[1].Type())
		}
		runtime, err := database.Runtime.Prepare(query.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteStatement{Runtime: runtime}
	}}
}
func SQLiteBeginBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteBegin expects 1 argument, got %d", len(args))
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqliteBegin")
		if errObj != nil {
			return errObj
		}
		runtime, err := database.Runtime.Begin()
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteTransaction{Runtime: runtime}
	}}
}
func SQLiteCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteClose expects 1 argument, got %d", len(args))
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqliteClose")
		if errObj != nil {
			return errObj
		}
		if err := database.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return &object.Boolean{Value: true}
	}}
}
func SQLiteIsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteIsOpen expects 1 argument, got %d", len(args))
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqliteIsOpen")
		if errObj != nil {
			return errObj
		}
		return &object.Boolean{Value: database.Runtime.IsOpen()}
	}}
}
func SQLitePathBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlitePath expects 1 argument, got %d", len(args))
		}
		database, errObj := sqliteDatabaseArg(args[0], "sqlitePath")
		if errObj != nil {
			return errObj
		}
		return &object.String{Value: database.Path}
	}}
}

func SQLiteStatementExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("sqliteStatementExec expects 1 or 2 arguments, got %d", len(args))
		}
		statement, errObj := sqliteStatementArg(args[0], "sqliteStatementExec")
		if errObj != nil {
			return errObj
		}
		var params []object.Object
		if len(args) == 2 {
			params, errObj = sqliteParams(args[1], "sqliteStatementExec")
			if errObj != nil {
				return errObj
			}
		}
		result, err := statement.Runtime.Exec(params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func SQLiteStatementQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("sqliteStatementQuery expects 1 or 2 arguments, got %d", len(args))
		}
		statement, errObj := sqliteStatementArg(args[0], "sqliteStatementQuery")
		if errObj != nil {
			return errObj
		}
		var params []object.Object
		if len(args) == 2 {
			params, errObj = sqliteParams(args[1], "sqliteStatementQuery")
			if errObj != nil {
				return errObj
			}
		}
		rows, err := statement.Runtime.Query(params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLiteStatementCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementClose expects 1 argument, got %d", len(args))
		}
		statement, errObj := sqliteStatementArg(args[0], "sqliteStatementClose")
		if errObj != nil {
			return errObj
		}
		if err := statement.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return &object.Boolean{Value: true}
	}}
}
func SQLiteStatementOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementOpen expects 1 argument, got %d", len(args))
		}
		statement, errObj := sqliteStatementArg(args[0], "sqliteStatementOpen")
		if errObj != nil {
			return errObj
		}
		return &object.Boolean{Value: statement.Runtime.IsOpen()}
	}}
}
func SQLiteStatementSQLBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementSQL expects 1 argument, got %d", len(args))
		}
		statement, errObj := sqliteStatementArg(args[0], "sqliteStatementSQL")
		if errObj != nil {
			return errObj
		}
		return &object.String{Value: statement.Runtime.SQL()}
	}}
}

func SQLiteTransactionExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, errObj := sqliteQueryArgs(args, "sqliteTransactionExec")
		if errObj != nil {
			return errObj
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteTransactionExec")
		if errObj != nil {
			return errObj
		}
		result, err := transaction.Runtime.Exec(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func SQLiteTransactionQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, errObj := sqliteQueryArgs(args, "sqliteTransactionQuery")
		if errObj != nil {
			return errObj
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteTransactionQuery")
		if errObj != nil {
			return errObj
		}
		rows, err := transaction.Runtime.Query(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLiteTransactionPrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sqliteTransactionPrepare expects 2 arguments, got %d", len(args))
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteTransactionPrepare")
		if errObj != nil {
			return errObj
		}
		query, ok := args[1].(*object.String)
		if !ok {
			return NewError("sqliteTransactionPrepare query must be STRING, got %s", args[1].Type())
		}
		runtime, err := transaction.Runtime.Prepare(query.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteStatement{Runtime: runtime}
	}}
}
func SQLiteCommitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteCommit expects 1 argument, got %d", len(args))
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteCommit")
		if errObj != nil {
			return errObj
		}
		if err := transaction.Runtime.Commit(); err != nil {
			return NewError("%s", err)
		}
		return &object.Boolean{Value: true}
	}}
}
func SQLiteRollbackBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteRollback expects 1 argument, got %d", len(args))
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteRollback")
		if errObj != nil {
			return errObj
		}
		if err := transaction.Runtime.Rollback(); err != nil {
			return NewError("%s", err)
		}
		return &object.Boolean{Value: true}
	}}
}
func SQLiteTransactionActiveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteTransactionActive expects 1 argument, got %d", len(args))
		}
		transaction, errObj := sqliteTransactionArg(args[0], "sqliteTransactionActive")
		if errObj != nil {
			return errObj
		}
		return &object.Boolean{Value: transaction.Runtime.Active()}
	}}
}

func prependSQLiteReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		all := make([]object.Object, 0, len(args)+1)
		all = append(all, receiver)
		all = append(all, args...)
		return builtin.Fn(all...)
	}}
}
func SQLiteDatabaseMethod(database *object.SQLiteDatabase, name string) object.Object {
	switch name {
	case "exec":
		return prependSQLiteReceiver(database, SQLiteExecBuiltin())
	case "query":
		return prependSQLiteReceiver(database, SQLiteQueryBuiltin())
	case "prepare":
		return prependSQLiteReceiver(database, SQLitePrepareBuiltin())
	case "begin":
		return prependSQLiteReceiver(database, SQLiteBeginBuiltin())
	case "close":
		return prependSQLiteReceiver(database, SQLiteCloseBuiltin())
	case "isOpen":
		return prependSQLiteReceiver(database, SQLiteIsOpenBuiltin())
	case "path":
		return prependSQLiteReceiver(database, SQLitePathBuiltin())
	default:
		return nil
	}
}
func SQLiteStatementMethod(statement *object.SQLiteStatement, name string) object.Object {
	switch name {
	case "exec":
		return prependSQLiteReceiver(statement, SQLiteStatementExecBuiltin())
	case "query":
		return prependSQLiteReceiver(statement, SQLiteStatementQueryBuiltin())
	case "close":
		return prependSQLiteReceiver(statement, SQLiteStatementCloseBuiltin())
	case "isOpen":
		return prependSQLiteReceiver(statement, SQLiteStatementOpenBuiltin())
	case "sql":
		return prependSQLiteReceiver(statement, SQLiteStatementSQLBuiltin())
	default:
		return nil
	}
}
func SQLiteTransactionMethod(transaction *object.SQLiteTransaction, name string) object.Object {
	switch name {
	case "exec":
		return prependSQLiteReceiver(transaction, SQLiteTransactionExecBuiltin())
	case "query":
		return prependSQLiteReceiver(transaction, SQLiteTransactionQueryBuiltin())
	case "prepare":
		return prependSQLiteReceiver(transaction, SQLiteTransactionPrepareBuiltin())
	case "commit":
		return prependSQLiteReceiver(transaction, SQLiteCommitBuiltin())
	case "rollback":
		return prependSQLiteReceiver(transaction, SQLiteRollbackBuiltin())
	case "active":
		return prependSQLiteReceiver(transaction, SQLiteTransactionActiveBuiltin())
	default:
		return nil
	}
}
