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
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"zumbra/binarydata"
	"zumbra/object"
)

const sqliteMemoryPath = ":memory:"

var sqliteIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

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

type sqliteRowsHandle struct {
	owner   *sqliteHandle
	tx      *sqliteTransactionHandle
	stmt    *C.sqlite3_stmt
	columns []string
	closed  bool
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

func emptySQLiteParams() object.Object { return &object.Array{Elements: nil} }

func (h *sqliteHandle) Exec(query string, params object.Object) (object.SQLiteExecResult, error) {
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

func (h *sqliteHandle) Query(query string, params object.Object) ([]map[string]object.Object, error) {
	rows, err := h.QueryStream(query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}

func (h *sqliteHandle) QueryStream(query string, params object.Object) (object.SQLRowsRuntime, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeTx {
		return nil, fmt.Errorf("SQLite database has an active transaction; use the transaction handle")
	}
	statement, err := h.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	if err := bindSQLiteParams(statement, params); err != nil {
		C.sqlite3_finalize(statement)
		return nil, err
	}
	return newSQLiteRowsHandle(h, nil, statement), nil
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
	if err := h.execSQLLocked("BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}
	h.activeTx = true
	return &sqliteTransactionHandle{owner: h, active: true}, nil
}

func (h *sqliteHandle) execSQLLocked(query string) error {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	var errorMessage *C.char
	code := C.sqlite3_exec(h.db, cQuery, nil, nil, &errorMessage)
	if code != C.SQLITE_OK {
		message := sqliteMessage(h.db)
		if errorMessage != nil {
			message = C.GoString(errorMessage)
			C.sqlite3_free(unsafe.Pointer(errorMessage))
		}
		return fmt.Errorf("execute SQLite SQL: %s", message)
	}
	return nil
}

func (h *sqliteHandle) execStatementLocked(statement *C.sqlite3_stmt, params object.Object) (object.SQLiteExecResult, error) {
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
	return object.SQLiteExecResult{LastInsertID: int64(C.sqlite3_last_insert_rowid(h.db)), RowsAffected: int64(C.sqlite3_changes64(h.db))}, nil
}

func collectSQLRows(rows object.SQLRowsRuntime) ([]map[string]object.Object, error) {
	result := make([]map[string]object.Object, 0)
	for {
		row, ok, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return result, nil
		}
		result = append(result, row)
	}
}

func bindSQLiteParams(statement *C.sqlite3_stmt, params object.Object) error {
	C.sqlite3_reset(statement)
	C.sqlite3_clear_bindings(statement)
	expected := int(C.sqlite3_bind_parameter_count(statement))
	if params == nil || params.Type() == object.NULL_OBJ {
		if expected != 0 {
			return fmt.Errorf("SQLite statement expects %d parameters, got 0", expected)
		}
		return nil
	}
	switch current := params.(type) {
	case *object.Array:
		if len(current.Elements) != expected {
			return fmt.Errorf("SQLite statement expects %d parameters, got %d", expected, len(current.Elements))
		}
		for index, value := range current.Elements {
			if err := bindSQLiteValue(statement, index+1, value); err != nil {
				return fmt.Errorf("bind SQLite parameter %d: %w", index+1, err)
			}
		}
		return nil
	case *object.Dict:
		values := make(map[string]object.Object, len(current.Pairs))
		for _, pair := range current.Pairs {
			key, ok := pair.Key.(*object.String)
			if !ok {
				return fmt.Errorf("SQLite named parameter keys must be strings")
			}
			values[key.Value] = pair.Value
		}
		used := make(map[string]bool, len(values))
		for index := 1; index <= expected; index++ {
			raw := C.sqlite3_bind_parameter_name(statement, C.int(index))
			if raw == nil {
				return fmt.Errorf("SQLite positional parameter %d requires array parameters", index)
			}
			name := C.GoString(raw)
			trimmed := strings.TrimLeft(name, ":@$?")
			value, ok := values[name]
			usedKey := name
			if !ok {
				value, ok = values[trimmed]
				usedKey = trimmed
			}
			if !ok {
				return fmt.Errorf("missing SQLite named parameter %s", name)
			}
			if err := bindSQLiteValue(statement, index, value); err != nil {
				return fmt.Errorf("bind SQLite parameter %s: %w", name, err)
			}
			used[usedKey] = true
		}
		for key := range values {
			if !used[key] && !used[strings.TrimLeft(key, ":@$?")] {
				return fmt.Errorf("unknown SQLite named parameter %s", key)
			}
		}
		return nil
	default:
		return fmt.Errorf("SQLite parameters must be an array or dictionary, got %s", params.Type())
	}
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

func newSQLiteRowsHandle(owner *sqliteHandle, tx *sqliteTransactionHandle, statement *C.sqlite3_stmt) *sqliteRowsHandle {
	count := int(C.sqlite3_column_count(statement))
	columns := make([]string, count)
	for index := 0; index < count; index++ {
		columns[index] = C.GoString(C.sqlite3_column_name(statement, C.int(index)))
	}
	return &sqliteRowsHandle{owner: owner, tx: tx, stmt: statement, columns: columns}
}

func (r *sqliteRowsHandle) Next() (map[string]object.Object, bool, error) {
	if r == nil || r.owner == nil {
		return nil, false, fmt.Errorf("SQLite rows are unavailable")
	}
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	if r.closed || r.stmt == nil {
		return nil, false, nil
	}
	if r.tx != nil && !r.tx.active {
		return nil, false, fmt.Errorf("SQLite transaction is no longer active")
	}
	code := C.sqlite3_step(r.stmt)
	if code == C.SQLITE_DONE {
		C.sqlite3_finalize(r.stmt)
		r.stmt = nil
		r.closed = true
		return nil, false, nil
	}
	if code != C.SQLITE_ROW {
		return nil, false, fmt.Errorf("query SQLite statement: %s", sqliteMessage(r.owner.db))
	}
	row := make(map[string]object.Object, len(r.columns))
	for index, name := range r.columns {
		row[name] = sqliteColumnObject(r.stmt, index)
	}
	return row, true, nil
}
func (r *sqliteRowsHandle) Columns() []string { return append([]string(nil), r.columns...) }
func (r *sqliteRowsHandle) Close() error {
	if r == nil || r.owner == nil {
		return nil
	}
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	if r.closed || r.stmt == nil {
		return nil
	}
	code := C.sqlite3_finalize(r.stmt)
	r.stmt = nil
	r.closed = true
	if code != C.SQLITE_OK {
		return fmt.Errorf("close SQLite rows: %s", sqliteMessage(r.owner.db))
	}
	return nil
}
func (r *sqliteRowsHandle) IsOpen() bool {
	if r == nil || r.owner == nil {
		return false
	}
	r.owner.mu.Lock()
	defer r.owner.mu.Unlock()
	return !r.closed && r.stmt != nil && r.owner.open
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
func (s *sqliteStatementHandle) Exec(params object.Object) (object.SQLiteExecResult, error) {
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
func (s *sqliteStatementHandle) Query(params object.Object) ([]map[string]object.Object, error) {
	rows, err := s.QueryStream(params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}
func (s *sqliteStatementHandle) QueryStream(params object.Object) (object.SQLRowsRuntime, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if s.closed || s.stmt == nil {
		return nil, fmt.Errorf("SQLite statement is closed")
	}
	duplicate, err := s.owner.prepareLocked(s.query)
	if err != nil {
		return nil, err
	}
	if err := bindSQLiteParams(duplicate, params); err != nil {
		C.sqlite3_finalize(duplicate)
		return nil, err
	}
	return newSQLiteRowsHandle(s.owner, s.tx, duplicate), nil
}
func (s *sqliteStatementHandle) ParameterCount() int {
	if s == nil || s.owner == nil {
		return 0
	}
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	if s.closed || s.stmt == nil {
		return 0
	}
	return int(C.sqlite3_bind_parameter_count(s.stmt))
}
func (s *sqliteStatementHandle) ColumnNames() []string {
	if s == nil || s.owner == nil {
		return nil
	}
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	if s.closed || s.stmt == nil {
		return nil
	}
	count := int(C.sqlite3_column_count(s.stmt))
	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = C.GoString(C.sqlite3_column_name(s.stmt, C.int(i)))
	}
	return result
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
func (t *sqliteTransactionHandle) Exec(query string, params object.Object) (object.SQLiteExecResult, error) {
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
func (t *sqliteTransactionHandle) Query(query string, params object.Object) ([]map[string]object.Object, error) {
	rows, err := t.QueryStream(query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}
func (t *sqliteTransactionHandle) QueryStream(query string, params object.Object) (object.SQLRowsRuntime, error) {
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
	if err := bindSQLiteParams(statement, params); err != nil {
		C.sqlite3_finalize(statement)
		return nil, err
	}
	return newSQLiteRowsHandle(t.owner, t, statement), nil
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
func validateSQLiteSavepoint(name string) error {
	if !sqliteIdentifier.MatchString(name) {
		return fmt.Errorf("invalid SQLite savepoint name %q", name)
	}
	return nil
}
func (t *sqliteTransactionHandle) control(command, name string) error {
	if err := validateSQLiteSavepoint(name); err != nil {
		return err
	}
	if t == nil || t.owner == nil {
		return fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return fmt.Errorf("SQLite transaction is no longer active")
	}
	return t.owner.execSQLLocked(command + " " + name)
}
func (t *sqliteTransactionHandle) Savepoint(name string) error { return t.control("SAVEPOINT", name) }
func (t *sqliteTransactionHandle) RollbackTo(name string) error {
	return t.control("ROLLBACK TO", name)
}
func (t *sqliteTransactionHandle) Release(name string) error { return t.control("RELEASE", name) }
func (t *sqliteTransactionHandle) finish(command string) error {
	if t == nil || t.owner == nil {
		return fmt.Errorf("SQLite transaction is unavailable")
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	if !t.active {
		return fmt.Errorf("SQLite transaction is no longer active")
	}
	err := t.owner.execSQLLocked(command)
	if err == nil {
		t.active = false
		t.owner.activeTx = false
	}
	return err
}
func (t *sqliteTransactionHandle) Commit() error   { return t.finish("COMMIT") }
func (t *sqliteTransactionHandle) Rollback() error { return t.finish("ROLLBACK") }

func (h *sqliteHandle) Migrate(migrations []object.SQLiteMigration) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeTx {
		return 0, fmt.Errorf("cannot migrate SQLite database with an active transaction")
	}
	if !h.open || h.db == nil {
		return 0, fmt.Errorf("SQLite database is closed")
	}
	if err := h.execSQLLocked(`CREATE TABLE IF NOT EXISTS _zumbra_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return 0, err
	}
	seen := map[int64]bool{}
	for _, migration := range migrations {
		if migration.Version <= 0 {
			return 0, fmt.Errorf("migration version must be positive")
		}
		if strings.TrimSpace(migration.SQL) == "" {
			return 0, fmt.Errorf("migration %d SQL is empty", migration.Version)
		}
		if seen[migration.Version] {
			return 0, fmt.Errorf("duplicate migration version %d", migration.Version)
		}
		seen[migration.Version] = true
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	applied := map[int64]bool{}
	statement, err := h.prepareLocked(`SELECT version FROM _zumbra_migrations`)
	if err != nil {
		return 0, err
	}
	for {
		code := C.sqlite3_step(statement)
		if code == C.SQLITE_DONE {
			break
		}
		if code != C.SQLITE_ROW {
			C.sqlite3_finalize(statement)
			return 0, fmt.Errorf("read SQLite migrations: %s", sqliteMessage(h.db))
		}
		applied[int64(C.sqlite3_column_int64(statement, 0))] = true
	}
	C.sqlite3_finalize(statement)
	if err := h.execSQLLocked("BEGIN IMMEDIATE"); err != nil {
		return 0, err
	}
	var count int64
	rollback := true
	defer func() {
		if rollback {
			_ = h.execSQLLocked("ROLLBACK")
		}
	}()
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := h.execSQLLocked(migration.SQL); err != nil {
			return count, fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		insert, err := h.prepareLocked(`INSERT INTO _zumbra_migrations(version, name, applied_at) VALUES (?, ?, ?)`)
		if err != nil {
			return count, err
		}
		params := &object.Array{Elements: []object.Object{&object.Integer{Value: migration.Version}, &object.String{Value: migration.Name}, &object.String{Value: time.Now().UTC().Format(time.RFC3339Nano)}}}
		_, err = h.execStatementLocked(insert, params)
		C.sqlite3_finalize(insert)
		if err != nil {
			return count, err
		}
		count++
	}
	if err := h.execSQLLocked("COMMIT"); err != nil {
		return count, err
	}
	rollback = false
	return count, nil
}
func (h *sqliteHandle) SchemaVersion() (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.open || h.db == nil {
		return 0, fmt.Errorf("SQLite database is closed")
	}
	exists, err := h.prepareLocked(`SELECT COALESCE(MAX(version), 0) FROM _zumbra_migrations`)
	if err != nil {
		return 0, nil
	}
	defer C.sqlite3_finalize(exists)
	if code := C.sqlite3_step(exists); code == C.SQLITE_ROW {
		return int64(C.sqlite3_column_int64(exists, 0)), nil
	}
	return 0, nil
}

func sqliteRowsObject(rows []map[string]object.Object) object.Object {
	elements := make([]object.Object, 0, len(rows))
	for _, row := range rows {
		elements = append(elements, sqliteRowObject(row))
	}
	return &object.Array{Elements: elements}
}
func sqliteRowObject(row map[string]object.Object) object.Object {
	pairs := make(map[object.DictKey]object.DictPair, len(row))
	for name, value := range row {
		key := &object.String{Value: name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: value}
	}
	return &object.Dict{Pairs: pairs}
}
func sqliteExecResultObject(result object.SQLiteExecResult) object.Object {
	pairs := map[object.DictKey]object.DictPair{}
	for name, value := range map[string]int64{"lastInsertId": result.LastInsertID, "rowsAffected": result.RowsAffected} {
		key := &object.String{Value: name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Integer{Value: value}}
	}
	return &object.Dict{Pairs: pairs}
}
func sqliteParamsObject(value object.Object, name string) (object.Object, *object.Error) {
	if value == nil || value.Type() == object.NULL_OBJ {
		return emptySQLiteParams(), nil
	}
	if value.Type() != object.ARRAY_OBJ && value.Type() != object.DICT_OBJ {
		return nil, NewError("%s parameters must be an array or dictionary, got %s", name, value.Type())
	}
	return value, nil
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
func sqlRowsArg(value object.Object, name string) (*object.SQLRows, *object.Error) {
	rows, ok := value.(*object.SQLRows)
	if !ok || rows.Runtime == nil {
		return nil, NewError("%s expects SQLRows, got %s", name, value.Type())
	}
	return rows, nil
}
func sqliteQueryArgs(args []object.Object, name string) (string, object.Object, *object.Error) {
	if len(args) < 2 || len(args) > 3 {
		return "", nil, NewError("%s expects 2 or 3 arguments, got %d", name, len(args))
	}
	query, ok := args[1].(*object.String)
	if !ok {
		return "", nil, NewError("%s query must be STRING, got %s", name, args[1].Type())
	}
	if len(args) == 2 {
		return query.Value, emptySQLiteParams(), nil
	}
	params, errObj := sqliteParamsObject(args[2], name)
	return query.Value, params, errObj
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
func SQLiteExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteExec")
		if e != nil {
			return e
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteExec")
		if e != nil {
			return e
		}
		result, err := db.Runtime.Exec(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func SQLiteQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteQuery")
		if e != nil {
			return e
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteQuery")
		if e != nil {
			return e
		}
		rows, err := db.Runtime.Query(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLiteQueryOneBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		result := SQLiteQueryBuiltin().Fn(args...)
		if result.Type() == object.ERROR_OBJ {
			return result
		}
		rows := result.(*object.Array)
		if len(rows.Elements) == 0 {
			return &object.Null{}
		}
		return rows.Elements[0]
	}}
}
func SQLiteQueryStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteQueryStream")
		if e != nil {
			return e
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteQueryStream")
		if e != nil {
			return e
		}
		rows, err := db.Runtime.QueryStream(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: rows, Driver: "sqlite"}
	}}
}
func SQLitePrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sqlitePrepare expects 2 arguments, got %d", len(args))
		}
		db, e := sqliteDatabaseArg(args[0], "sqlitePrepare")
		if e != nil {
			return e
		}
		query, ok := args[1].(*object.String)
		if !ok {
			return NewError("sqlitePrepare query must be STRING, got %s", args[1].Type())
		}
		runtime, err := db.Runtime.Prepare(query.Value)
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
		db, e := sqliteDatabaseArg(args[0], "sqliteBegin")
		if e != nil {
			return e
		}
		runtime, err := db.Runtime.Begin()
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
		db, e := sqliteDatabaseArg(args[0], "sqliteClose")
		if e != nil {
			return e
		}
		if err := db.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLiteIsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteIsOpen expects 1 argument, got %d", len(args))
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteIsOpen")
		if e != nil {
			return e
		}
		return NewBoolean(db.Runtime.IsOpen())
	}}
}
func SQLitePathBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlitePath expects 1 argument, got %d", len(args))
		}
		db, e := sqliteDatabaseArg(args[0], "sqlitePath")
		if e != nil {
			return e
		}
		return &object.String{Value: db.Path}
	}}
}
func parseSQLiteMigrations(value object.Object) ([]object.SQLiteMigration, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("sqliteMigrate migrations must be an array")
	}
	migrations := make([]object.SQLiteMigration, 0, len(array.Elements))
	for index, element := range array.Elements {
		dict, ok := element.(*object.Dict)
		if !ok {
			return nil, NewError("sqliteMigrate migration %d must be a dictionary", index)
		}
		var migration object.SQLiteMigration
		var haveVersion, haveSQL bool
		for _, pair := range dict.Pairs {
			key, ok := pair.Key.(*object.String)
			if !ok {
				continue
			}
			switch key.Value {
			case "version":
				v, ok := pair.Value.(*object.Integer)
				if !ok {
					return nil, NewError("migration version must be int")
				}
				migration.Version = v.Value
				haveVersion = true
			case "name":
				v, ok := pair.Value.(*object.String)
				if !ok {
					return nil, NewError("migration name must be string")
				}
				migration.Name = v.Value
			case "sql":
				v, ok := pair.Value.(*object.String)
				if !ok {
					return nil, NewError("migration sql must be string")
				}
				migration.SQL = v.Value
				haveSQL = true
			}
		}
		if !haveVersion || !haveSQL {
			return nil, NewError("migration %d requires version and sql", index)
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}
func SQLiteMigrateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sqliteMigrate expects 2 arguments, got %d", len(args))
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteMigrate")
		if e != nil {
			return e
		}
		migrations, e := parseSQLiteMigrations(args[1])
		if e != nil {
			return e
		}
		count, err := db.Runtime.Migrate(migrations)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: count}
	}}
}
func SQLiteSchemaVersionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteSchemaVersion expects 1 argument")
		}
		db, e := sqliteDatabaseArg(args[0], "sqliteSchemaVersion")
		if e != nil {
			return e
		}
		version, err := db.Runtime.SchemaVersion()
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: version}
	}}
}

func statementParams(args []object.Object, name string) (object.Object, *object.Error) {
	if len(args) == 1 {
		return emptySQLiteParams(), nil
	}
	if len(args) != 2 {
		return nil, NewError("%s expects 1 or 2 arguments, got %d", name, len(args))
	}
	return sqliteParamsObject(args[1], name)
}
func SQLiteStatementExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("sqliteStatementExec expects 1 or 2 arguments, got %d", len(args))
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementExec")
		if e != nil {
			return e
		}
		params, e := statementParams(args, "sqliteStatementExec")
		if e != nil {
			return e
		}
		result, err := st.Runtime.Exec(params)
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
		st, e := sqliteStatementArg(args[0], "sqliteStatementQuery")
		if e != nil {
			return e
		}
		params, e := statementParams(args, "sqliteStatementQuery")
		if e != nil {
			return e
		}
		rows, err := st.Runtime.Query(params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLiteStatementQueryStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("sqliteStatementQueryStream expects 1 or 2 arguments")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementQueryStream")
		if e != nil {
			return e
		}
		params, e := statementParams(args, "sqliteStatementQueryStream")
		if e != nil {
			return e
		}
		rows, err := st.Runtime.QueryStream(params)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: rows, Driver: "sqlite"}
	}}
}
func SQLiteStatementCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementClose expects 1 argument")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementClose")
		if e != nil {
			return e
		}
		if err := st.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLiteStatementOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementOpen expects 1 argument")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementOpen")
		if e != nil {
			return e
		}
		return NewBoolean(st.Runtime.IsOpen())
	}}
}
func SQLiteStatementSQLBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementSQL expects 1 argument")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementSQL")
		if e != nil {
			return e
		}
		return &object.String{Value: st.Runtime.SQL()}
	}}
}
func SQLiteStatementParameterCountBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementParameterCount expects 1 argument")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementParameterCount")
		if e != nil {
			return e
		}
		return &object.Integer{Value: int64(st.Runtime.ParameterCount())}
	}}
}
func SQLiteStatementColumnsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteStatementColumns expects 1 argument")
		}
		st, e := sqliteStatementArg(args[0], "sqliteStatementColumns")
		if e != nil {
			return e
		}
		items := []object.Object{}
		for _, v := range st.Runtime.ColumnNames() {
			items = append(items, &object.String{Value: v})
		}
		return &object.Array{Elements: items}
	}}
}

func SQLiteTransactionExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteTransactionExec")
		if e != nil {
			return e
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteTransactionExec")
		if e != nil {
			return e
		}
		result, err := tx.Runtime.Exec(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func SQLiteTransactionQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteTransactionQuery")
		if e != nil {
			return e
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteTransactionQuery")
		if e != nil {
			return e
		}
		rows, err := tx.Runtime.Query(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func SQLiteTransactionQueryStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		query, params, e := sqliteQueryArgs(args, "sqliteTransactionQueryStream")
		if e != nil {
			return e
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteTransactionQueryStream")
		if e != nil {
			return e
		}
		rows, err := tx.Runtime.QueryStream(query, params)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: rows, Driver: "sqlite"}
	}}
}
func SQLiteTransactionPrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sqliteTransactionPrepare expects 2 arguments")
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteTransactionPrepare")
		if e != nil {
			return e
		}
		q, ok := args[1].(*object.String)
		if !ok {
			return NewError("query must be string")
		}
		runtime, err := tx.Runtime.Prepare(q.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLiteStatement{Runtime: runtime}
	}}
}
func sqliteTxControlBuiltin(name string, action func(object.SQLiteTransactionRuntime, string) error) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments", name)
		}
		tx, e := sqliteTransactionArg(args[0], name)
		if e != nil {
			return e
		}
		n, ok := args[1].(*object.String)
		if !ok {
			return NewError("%s name must be string", name)
		}
		if err := action(tx.Runtime, n.Value); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLiteSavepointBuiltin() *object.Builtin {
	return sqliteTxControlBuiltin("sqliteSavepoint", func(t object.SQLiteTransactionRuntime, n string) error { return t.Savepoint(n) })
}
func SQLiteRollbackToBuiltin() *object.Builtin {
	return sqliteTxControlBuiltin("sqliteRollbackTo", func(t object.SQLiteTransactionRuntime, n string) error { return t.RollbackTo(n) })
}
func SQLiteReleaseBuiltin() *object.Builtin {
	return sqliteTxControlBuiltin("sqliteRelease", func(t object.SQLiteTransactionRuntime, n string) error { return t.Release(n) })
}
func SQLiteCommitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteCommit expects 1 argument")
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteCommit")
		if e != nil {
			return e
		}
		if err := tx.Runtime.Commit(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLiteRollbackBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteRollback expects 1 argument")
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteRollback")
		if e != nil {
			return e
		}
		if err := tx.Runtime.Rollback(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLiteTransactionActiveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqliteTransactionActive expects 1 argument")
		}
		tx, e := sqliteTransactionArg(args[0], "sqliteTransactionActive")
		if e != nil {
			return e
		}
		return NewBoolean(tx.Runtime.Active())
	}}
}

func SQLRowsNextBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlRowsNext expects 1 argument")
		}
		rows, e := sqlRowsArg(args[0], "sqlRowsNext")
		if e != nil {
			return e
		}
		row, ok, err := rows.Runtime.Next()
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Array{Elements: []object.Object{func() object.Object {
			if !ok {
				return &object.Null{}
			}
			return sqliteRowObject(row)
		}(), NewBoolean(ok)}}
	}}
}
func SQLRowsColumnsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlRowsColumns expects 1 argument")
		}
		rows, e := sqlRowsArg(args[0], "sqlRowsColumns")
		if e != nil {
			return e
		}
		items := []object.Object{}
		for _, v := range rows.Runtime.Columns() {
			items = append(items, &object.String{Value: v})
		}
		return &object.Array{Elements: items}
	}}
}
func SQLRowsCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlRowsClose expects 1 argument")
		}
		rows, e := sqlRowsArg(args[0], "sqlRowsClose")
		if e != nil {
			return e
		}
		if err := rows.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SQLRowsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sqlRowsOpen expects 1 argument")
		}
		rows, e := sqlRowsArg(args[0], "sqlRowsOpen")
		if e != nil {
			return e
		}
		return NewBoolean(rows.Runtime.IsOpen())
	}}
}

func prependSQLiteReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		all := append([]object.Object{receiver}, args...)
		return builtin.Fn(all...)
	}}
}
func SQLiteDatabaseMethod(database *object.SQLiteDatabase, name string) object.Object {
	switch name {
	case "exec":
		return prependSQLiteReceiver(database, SQLiteExecBuiltin())
	case "query":
		return prependSQLiteReceiver(database, SQLiteQueryBuiltin())
	case "queryOne":
		return prependSQLiteReceiver(database, SQLiteQueryOneBuiltin())
	case "stream":
		return prependSQLiteReceiver(database, SQLiteQueryStreamBuiltin())
	case "prepare":
		return prependSQLiteReceiver(database, SQLitePrepareBuiltin())
	case "begin":
		return prependSQLiteReceiver(database, SQLiteBeginBuiltin())
	case "migrate":
		return prependSQLiteReceiver(database, SQLiteMigrateBuiltin())
	case "schemaVersion":
		return prependSQLiteReceiver(database, SQLiteSchemaVersionBuiltin())
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
	case "stream":
		return prependSQLiteReceiver(statement, SQLiteStatementQueryStreamBuiltin())
	case "parameterCount":
		return prependSQLiteReceiver(statement, SQLiteStatementParameterCountBuiltin())
	case "columns":
		return prependSQLiteReceiver(statement, SQLiteStatementColumnsBuiltin())
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
	case "stream":
		return prependSQLiteReceiver(transaction, SQLiteTransactionQueryStreamBuiltin())
	case "prepare":
		return prependSQLiteReceiver(transaction, SQLiteTransactionPrepareBuiltin())
	case "savepoint":
		return prependSQLiteReceiver(transaction, SQLiteSavepointBuiltin())
	case "rollbackTo":
		return prependSQLiteReceiver(transaction, SQLiteRollbackToBuiltin())
	case "release":
		return prependSQLiteReceiver(transaction, SQLiteReleaseBuiltin())
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
func SQLRowsMethod(rows *object.SQLRows, name string) object.Object {
	switch name {
	case "next":
		return prependSQLiteReceiver(rows, SQLRowsNextBuiltin())
	case "columns":
		return prependSQLiteReceiver(rows, SQLRowsColumnsBuiltin())
	case "close":
		return prependSQLiteReceiver(rows, SQLRowsCloseBuiltin())
	case "isOpen":
		return prependSQLiteReceiver(rows, SQLRowsOpenBuiltin())
	default:
		return nil
	}
}
