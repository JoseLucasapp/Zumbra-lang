package builtins

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sync"
	"time"

	"zumbra/object"

	_ "github.com/lib/pq"
)

var postgresSavepointIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var legacyPostgresMu sync.Mutex
var legacyPostgres *object.PostgresDatabase

type postgresHandle struct {
	mu   sync.RWMutex
	db   *sql.DB
	open bool
	dsn  string
}

type postgresStatementHandle struct {
	owner     *postgresHandle
	tx        *postgresTransactionHandle
	statement *sql.Stmt
	query     string
	mu        sync.RWMutex
	open      bool
}

type postgresTransactionHandle struct {
	owner       *postgresHandle
	transaction *sql.Tx
	mu          sync.RWMutex
	active      bool
}

type sqlRowsHandle struct {
	rows    *sql.Rows
	columns []string
	mu      sync.Mutex
	open    bool
}

func postgresObjects(params []object.Object) []any {
	values := make([]any, len(params))
	for index, value := range params {
		values[index] = objectToGoValue(value)
	}
	return values
}

func postgresParams(value object.Object, name string) ([]object.Object, *object.Error) {
	if value == nil || value.Type() == object.NULL_OBJ {
		return nil, nil
	}
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("%s parameters must be an array, got %s", name, value.Type())
	}
	return array.Elements, nil
}

func openPostgres(dsn string, timeout time.Duration) (*postgresHandle, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL database: %w", err)
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect PostgreSQL database: %w", err)
	}
	return &postgresHandle{db: db, open: true, dsn: dsn}, nil
}

func (h *postgresHandle) IsOpen() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.open && h.db != nil
}
func (h *postgresHandle) Ping() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.open || h.db == nil {
		return fmt.Errorf("PostgreSQL database is closed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.db.PingContext(ctx)
}
func (h *postgresHandle) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.open || h.db == nil {
		return nil
	}
	err := h.db.Close()
	if err == nil {
		h.db = nil
		h.open = false
	}
	return err
}
func (h *postgresHandle) ConfigurePool(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.db == nil {
		return
	}
	h.db.SetMaxOpenConns(maxOpen)
	h.db.SetMaxIdleConns(maxIdle)
	h.db.SetConnMaxLifetime(maxLifetime)
	h.db.SetConnMaxIdleTime(maxIdleTime)
}
func (h *postgresHandle) PoolStats() map[string]int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.db == nil {
		return map[string]int64{}
	}
	s := h.db.Stats()
	return map[string]int64{"maxOpen": int64(s.MaxOpenConnections), "open": int64(s.OpenConnections), "inUse": int64(s.InUse), "idle": int64(s.Idle), "waitCount": s.WaitCount, "waitDurationMs": s.WaitDuration.Milliseconds(), "maxIdleClosed": s.MaxIdleClosed, "maxIdleTimeClosed": s.MaxIdleTimeClosed, "maxLifetimeClosed": s.MaxLifetimeClosed}
}
func (h *postgresHandle) Exec(query string, params []object.Object) (object.SQLiteExecResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.open || h.db == nil {
		return object.SQLiteExecResult{}, fmt.Errorf("PostgreSQL database is closed")
	}
	result, err := h.db.Exec(query, postgresObjects(params)...)
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	rows, _ := result.RowsAffected()
	id, _ := result.LastInsertId()
	return object.SQLiteExecResult{LastInsertID: id, RowsAffected: rows}, nil
}
func (h *postgresHandle) QueryStream(query string, params []object.Object) (object.SQLRowsRuntime, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.open || h.db == nil {
		return nil, fmt.Errorf("PostgreSQL database is closed")
	}
	rows, err := h.db.Query(query, postgresObjects(params)...)
	if err != nil {
		return nil, err
	}
	return newSQLRowsHandle(rows)
}
func (h *postgresHandle) Query(query string, params []object.Object) ([]map[string]object.Object, error) {
	rows, err := h.QueryStream(query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}
func (h *postgresHandle) Prepare(query string) (object.PostgresStatementRuntime, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.open || h.db == nil {
		return nil, fmt.Errorf("PostgreSQL database is closed")
	}
	st, err := h.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &postgresStatementHandle{owner: h, statement: st, query: query, open: true}, nil
}
func (h *postgresHandle) Begin() (object.PostgresTransactionRuntime, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if !h.open || h.db == nil {
		return nil, fmt.Errorf("PostgreSQL database is closed")
	}
	tx, err := h.db.Begin()
	if err != nil {
		return nil, err
	}
	return &postgresTransactionHandle{owner: h, transaction: tx, active: true}, nil
}

func newSQLRowsHandle(rows *sql.Rows) (*sqlRowsHandle, error) {
	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, err
	}
	return &sqlRowsHandle{rows: rows, columns: columns, open: true}, nil
}
func (r *sqlRowsHandle) Next() (map[string]object.Object, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.open || r.rows == nil {
		return nil, false, nil
	}
	if !r.rows.Next() {
		err := r.rows.Err()
		r.rows.Close()
		r.rows = nil
		r.open = false
		return nil, false, err
	}
	values := make([]any, len(r.columns))
	pointers := make([]any, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	if err := r.rows.Scan(pointers...); err != nil {
		return nil, false, err
	}
	row := make(map[string]object.Object, len(values))
	for i, name := range r.columns {
		row[name] = goValueToObject(values[i])
	}
	return row, true, nil
}
func (r *sqlRowsHandle) Columns() []string { return append([]string(nil), r.columns...) }
func (r *sqlRowsHandle) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.open || r.rows == nil {
		return nil
	}
	err := r.rows.Close()
	r.rows = nil
	r.open = false
	return err
}
func (r *sqlRowsHandle) IsOpen() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.open && r.rows != nil
}

func (s *postgresStatementHandle) IsOpen() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.open && s.statement != nil && s.owner.IsOpen()
}
func (s *postgresStatementHandle) SQL() string { return s.query }
func (s *postgresStatementHandle) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open || s.statement == nil {
		return nil
	}
	err := s.statement.Close()
	if err == nil {
		s.statement = nil
		s.open = false
	}
	return err
}
func (s *postgresStatementHandle) Exec(params []object.Object) (object.SQLiteExecResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.open || s.statement == nil {
		return object.SQLiteExecResult{}, fmt.Errorf("PostgreSQL statement is closed")
	}
	result, err := s.statement.Exec(postgresObjects(params)...)
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	rows, _ := result.RowsAffected()
	id, _ := result.LastInsertId()
	return object.SQLiteExecResult{LastInsertID: id, RowsAffected: rows}, nil
}
func (s *postgresStatementHandle) QueryStream(params []object.Object) (object.SQLRowsRuntime, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.open || s.statement == nil {
		return nil, fmt.Errorf("PostgreSQL statement is closed")
	}
	rows, err := s.statement.Query(postgresObjects(params)...)
	if err != nil {
		return nil, err
	}
	return newSQLRowsHandle(rows)
}
func (s *postgresStatementHandle) Query(params []object.Object) ([]map[string]object.Object, error) {
	rows, err := s.QueryStream(params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}

func (t *postgresTransactionHandle) Active() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active && t.transaction != nil
}
func (t *postgresTransactionHandle) Exec(query string, params []object.Object) (object.SQLiteExecResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.active || t.transaction == nil {
		return object.SQLiteExecResult{}, fmt.Errorf("PostgreSQL transaction is no longer active")
	}
	result, err := t.transaction.Exec(query, postgresObjects(params)...)
	if err != nil {
		return object.SQLiteExecResult{}, err
	}
	rows, _ := result.RowsAffected()
	id, _ := result.LastInsertId()
	return object.SQLiteExecResult{LastInsertID: id, RowsAffected: rows}, nil
}
func (t *postgresTransactionHandle) QueryStream(query string, params []object.Object) (object.SQLRowsRuntime, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.active || t.transaction == nil {
		return nil, fmt.Errorf("PostgreSQL transaction is no longer active")
	}
	rows, err := t.transaction.Query(query, postgresObjects(params)...)
	if err != nil {
		return nil, err
	}
	return newSQLRowsHandle(rows)
}
func (t *postgresTransactionHandle) Query(query string, params []object.Object) ([]map[string]object.Object, error) {
	rows, err := t.QueryStream(query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSQLRows(rows)
}
func (t *postgresTransactionHandle) Prepare(query string) (object.PostgresStatementRuntime, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.active || t.transaction == nil {
		return nil, fmt.Errorf("PostgreSQL transaction is no longer active")
	}
	st, err := t.transaction.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &postgresStatementHandle{owner: t.owner, tx: t, statement: st, query: query, open: true}, nil
}
func validatePostgresSavepoint(name string) error {
	if !postgresSavepointIdentifier.MatchString(name) {
		return fmt.Errorf("invalid PostgreSQL savepoint name %q", name)
	}
	return nil
}
func (t *postgresTransactionHandle) control(command, name string) error {
	if err := validatePostgresSavepoint(name); err != nil {
		return err
	}
	_, err := t.Exec(command+" "+name, nil)
	return err
}
func (t *postgresTransactionHandle) Savepoint(name string) error { return t.control("SAVEPOINT", name) }
func (t *postgresTransactionHandle) RollbackTo(name string) error {
	return t.control("ROLLBACK TO SAVEPOINT", name)
}
func (t *postgresTransactionHandle) Release(name string) error {
	return t.control("RELEASE SAVEPOINT", name)
}
func (t *postgresTransactionHandle) finish(commit bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active || t.transaction == nil {
		return fmt.Errorf("PostgreSQL transaction is no longer active")
	}
	var err error
	if commit {
		err = t.transaction.Commit()
	} else {
		err = t.transaction.Rollback()
	}
	if err == nil {
		t.active = false
		t.transaction = nil
	}
	return err
}
func (t *postgresTransactionHandle) Commit() error   { return t.finish(true) }
func (t *postgresTransactionHandle) Rollback() error { return t.finish(false) }

func postgresDatabaseArg(value object.Object, name string) (*object.PostgresDatabase, *object.Error) {
	db, ok := value.(*object.PostgresDatabase)
	if !ok || db.Runtime == nil {
		return nil, NewError("%s expects PostgresDatabase, got %s", name, value.Type())
	}
	return db, nil
}
func postgresStatementArg(value object.Object, name string) (*object.PostgresStatement, *object.Error) {
	st, ok := value.(*object.PostgresStatement)
	if !ok || st.Runtime == nil {
		return nil, NewError("%s expects PostgresStatement, got %s", name, value.Type())
	}
	return st, nil
}
func postgresTransactionArg(value object.Object, name string) (*object.PostgresTransaction, *object.Error) {
	tx, ok := value.(*object.PostgresTransaction)
	if !ok || tx.Runtime == nil {
		return nil, NewError("%s expects PostgresTransaction, got %s", name, value.Type())
	}
	return tx, nil
}
func postgresQueryArgs(args []object.Object, name string) (string, []object.Object, *object.Error) {
	if len(args) < 2 || len(args) > 3 {
		return "", nil, NewError("%s expects 2 or 3 arguments, got %d", name, len(args))
	}
	q, ok := args[1].(*object.String)
	if !ok {
		return "", nil, NewError("%s query must be string", name)
	}
	if len(args) == 2 {
		return q.Value, nil, nil
	}
	p, e := postgresParams(args[2], name)
	return q.Value, p, e
}
func postgresStatsObject(stats map[string]int64) object.Object {
	pairs := map[object.DictKey]object.DictPair{}
	for name, value := range stats {
		key := &object.String{Value: name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Integer{Value: value}}
	}
	return &object.Dict{Pairs: pairs}
}

func PostgresOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresOpen expects dsn and optional options")
		}
		dsn, ok := args[0].(*object.String)
		if !ok {
			return NewError("postgresOpen dsn must be string")
		}
		maxOpen, maxIdle := int64(0), int64(0)
		maxLifetime, maxIdleTime := int64(0), int64(0)
		timeout := int64(5000)
		if len(args) == 2 {
			options, e := objectDictMap(args[1], "postgresOpen options")
			if e != nil {
				return e
			}
			for name, target := range map[string]*int64{
				"maxOpen": &maxOpen, "maxIdle": &maxIdle,
				"maxLifetimeMs": &maxLifetime, "maxIdleTimeMs": &maxIdleTime,
				"timeoutMs": &timeout,
			} {
				if value, exists := options[name]; exists {
					parsed, parseError := concurrencyInt(value, "postgresOpen "+name)
					if parseError != nil {
						return parseError
					}
					*target = parsed
				}
			}
		}
		runtime, err := openPostgres(dsn.Value, time.Duration(timeout)*time.Millisecond)
		if err != nil {
			return NewError("%s", err)
		}
		runtime.ConfigurePool(int(maxOpen), int(maxIdle), time.Duration(maxLifetime)*time.Millisecond, time.Duration(maxIdleTime)*time.Millisecond)
		return &object.PostgresDatabase{Runtime: runtime, DSN: dsn.Value}
	}}
}
func PostgresConfigurePoolBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 5 {
			return NewError("postgresConfigurePool expects database, maxOpen, maxIdle, maxLifetimeMs, maxIdleTimeMs")
		}
		db, e := postgresDatabaseArg(args[0], "postgresConfigurePool")
		if e != nil {
			return e
		}
		values := make([]int64, 4)
		for i := range values {
			v, er := concurrencyInt(args[i+1], "postgresConfigurePool")
			if er != nil {
				return er
			}
			values[i] = v
		}
		db.Runtime.ConfigurePool(int(values[0]), int(values[1]), time.Duration(values[2])*time.Millisecond, time.Duration(values[3])*time.Millisecond)
		return db
	}}
}
func PostgresPoolStatsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresPoolStats expects 1 argument")
		}
		db, e := postgresDatabaseArg(args[0], "postgresPoolStats")
		if e != nil {
			return e
		}
		return postgresStatsObject(db.Runtime.PoolStats())
	}}
}
func PostgresPingBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresPing expects 1 argument")
		}
		db, e := postgresDatabaseArg(args[0], "postgresPing")
		if e != nil {
			return e
		}
		if err := db.Runtime.Ping(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresClose expects 1 argument")
		}
		db, e := postgresDatabaseArg(args[0], "postgresClose")
		if e != nil {
			return e
		}
		if err := db.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresIsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresIsOpen expects 1 argument")
		}
		db, e := postgresDatabaseArg(args[0], "postgresIsOpen")
		if e != nil {
			return e
		}
		return NewBoolean(db.Runtime.IsOpen())
	}}
}
func PostgresExecObjectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresQueryArgs(args, "postgresExecDb")
		if e != nil {
			return e
		}
		db, e := postgresDatabaseArg(args[0], "postgresExecDb")
		if e != nil {
			return e
		}
		result, err := db.Runtime.Exec(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(result)
	}}
}
func PostgresQueryObjectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresQueryArgs(args, "postgresQueryDb")
		if e != nil {
			return e
		}
		db, e := postgresDatabaseArg(args[0], "postgresQueryDb")
		if e != nil {
			return e
		}
		rows, err := db.Runtime.Query(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(rows)
	}}
}
func PostgresQueryOneBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		value := PostgresQueryObjectBuiltin().Fn(args...)
		if value.Type() == object.ERROR_OBJ {
			return value
		}
		rows := value.(*object.Array)
		if len(rows.Elements) == 0 {
			return &object.Null{}
		}
		return rows.Elements[0]
	}}
}
func PostgresQueryStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresQueryArgs(args, "postgresQueryStream")
		if e != nil {
			return e
		}
		db, e := postgresDatabaseArg(args[0], "postgresQueryStream")
		if e != nil {
			return e
		}
		rows, err := db.Runtime.QueryStream(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: rows, Driver: "postgres"}
	}}
}
func PostgresPrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("postgresPrepare expects 2 arguments")
		}
		db, e := postgresDatabaseArg(args[0], "postgresPrepare")
		if e != nil {
			return e
		}
		q, ok := args[1].(*object.String)
		if !ok {
			return NewError("postgresPrepare query must be string")
		}
		st, err := db.Runtime.Prepare(q.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.PostgresStatement{Runtime: st}
	}}
}
func PostgresBeginBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresBegin expects 1 argument")
		}
		db, e := postgresDatabaseArg(args[0], "postgresBegin")
		if e != nil {
			return e
		}
		tx, err := db.Runtime.Begin()
		if err != nil {
			return NewError("%s", err)
		}
		return &object.PostgresTransaction{Runtime: tx}
	}}
}
func postgresStatementParams(args []object.Object, name string) ([]object.Object, *object.Error) {
	if len(args) == 1 {
		return nil, nil
	}
	if len(args) != 2 {
		return nil, NewError("%s expects 1 or 2 arguments", name)
	}
	return postgresParams(args[1], name)
}
func PostgresStatementExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresStatementExec expects 1 or 2 arguments")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementExec")
		if e != nil {
			return e
		}
		p, e := postgresStatementParams(args, "postgresStatementExec")
		if e != nil {
			return e
		}
		r, err := st.Runtime.Exec(p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(r)
	}}
}
func PostgresStatementQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresStatementQuery expects 1 or 2 arguments")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementQuery")
		if e != nil {
			return e
		}
		p, e := postgresStatementParams(args, "postgresStatementQuery")
		if e != nil {
			return e
		}
		r, err := st.Runtime.Query(p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(r)
	}}
}
func PostgresStatementStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresStatementStream expects 1 or 2 arguments")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementStream")
		if e != nil {
			return e
		}
		p, e := postgresStatementParams(args, "postgresStatementStream")
		if e != nil {
			return e
		}
		r, err := st.Runtime.QueryStream(p)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: r, Driver: "postgres"}
	}}
}
func PostgresStatementCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresStatementClose expects 1 argument")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementClose")
		if e != nil {
			return e
		}
		if err := st.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresStatementOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresStatementOpen expects 1 argument")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementOpen")
		if e != nil {
			return e
		}
		return NewBoolean(st.Runtime.IsOpen())
	}}
}
func PostgresStatementSQLBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresStatementSQL expects 1 argument")
		}
		st, e := postgresStatementArg(args[0], "postgresStatementSQL")
		if e != nil {
			return e
		}
		return &object.String{Value: st.Runtime.SQL()}
	}}
}
func postgresTxQueryArgs(args []object.Object, name string) (string, []object.Object, *object.Error) {
	return postgresQueryArgs(args, name)
}
func PostgresTransactionExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresTxQueryArgs(args, "postgresTransactionExec")
		if e != nil {
			return e
		}
		tx, e := postgresTransactionArg(args[0], "postgresTransactionExec")
		if e != nil {
			return e
		}
		r, err := tx.Runtime.Exec(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteExecResultObject(r)
	}}
}
func PostgresTransactionQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresTxQueryArgs(args, "postgresTransactionQuery")
		if e != nil {
			return e
		}
		tx, e := postgresTransactionArg(args[0], "postgresTransactionQuery")
		if e != nil {
			return e
		}
		r, err := tx.Runtime.Query(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return sqliteRowsObject(r)
	}}
}
func PostgresTransactionStreamBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		q, p, e := postgresTxQueryArgs(args, "postgresTransactionStream")
		if e != nil {
			return e
		}
		tx, e := postgresTransactionArg(args[0], "postgresTransactionStream")
		if e != nil {
			return e
		}
		r, err := tx.Runtime.QueryStream(q, p)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SQLRows{Runtime: r, Driver: "postgres"}
	}}
}
func PostgresTransactionPrepareBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("postgresTransactionPrepare expects 2 arguments")
		}
		tx, e := postgresTransactionArg(args[0], "postgresTransactionPrepare")
		if e != nil {
			return e
		}
		q, ok := args[1].(*object.String)
		if !ok {
			return NewError("query must be string")
		}
		st, err := tx.Runtime.Prepare(q.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.PostgresStatement{Runtime: st}
	}}
}
func postgresTxControl(name string, action func(object.PostgresTransactionRuntime, string) error) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments", name)
		}
		tx, e := postgresTransactionArg(args[0], name)
		if e != nil {
			return e
		}
		n, ok := args[1].(*object.String)
		if !ok {
			return NewError("name must be string")
		}
		if err := action(tx.Runtime, n.Value); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresSavepointBuiltin() *object.Builtin {
	return postgresTxControl("postgresSavepoint", func(t object.PostgresTransactionRuntime, n string) error { return t.Savepoint(n) })
}
func PostgresRollbackToBuiltin() *object.Builtin {
	return postgresTxControl("postgresRollbackTo", func(t object.PostgresTransactionRuntime, n string) error { return t.RollbackTo(n) })
}
func PostgresReleaseBuiltin() *object.Builtin {
	return postgresTxControl("postgresRelease", func(t object.PostgresTransactionRuntime, n string) error { return t.Release(n) })
}
func PostgresCommitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresCommit expects 1 argument")
		}
		tx, e := postgresTransactionArg(args[0], "postgresCommit")
		if e != nil {
			return e
		}
		if err := tx.Runtime.Commit(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresRollbackBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresRollback expects 1 argument")
		}
		tx, e := postgresTransactionArg(args[0], "postgresRollback")
		if e != nil {
			return e
		}
		if err := tx.Runtime.Rollback(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func PostgresTransactionActiveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("postgresTransactionActive expects 1 argument")
		}
		tx, e := postgresTransactionArg(args[0], "postgresTransactionActive")
		if e != nil {
			return e
		}
		return NewBoolean(tx.Runtime.Active())
	}}
}

// Legacy global API remains compatible but delegates to the safe object API.
func PostgresConnectionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		value := PostgresOpenBuiltin().Fn(args...)
		if value.Type() == object.ERROR_OBJ {
			return value
		}
		legacyPostgresMu.Lock()
		if legacyPostgres != nil {
			_ = legacyPostgres.Runtime.Close()
		}
		legacyPostgres = value.(*object.PostgresDatabase)
		legacyPostgresMu.Unlock()
		return &object.Null{}
	}}
}
func PostgresExecBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresExec expects query and optional params")
		}
		legacyPostgresMu.Lock()
		db := legacyPostgres
		legacyPostgresMu.Unlock()
		if db == nil {
			return NewError("postgres is not connected; use postgresConnection or postgresOpen")
		}
		forward := []object.Object{db, args[0]}
		if len(args) == 2 {
			forward = append(forward, args[1])
		}
		return PostgresExecObjectBuiltin().Fn(forward...)
	}}
}
func PostgresQueryBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("postgresQuery expects query and optional params")
		}
		legacyPostgresMu.Lock()
		db := legacyPostgres
		legacyPostgresMu.Unlock()
		if db == nil {
			return NewError("postgres is not connected; use postgresConnection or postgresOpen")
		}
		forward := []object.Object{db, args[0]}
		if len(args) == 2 {
			forward = append(forward, args[1])
		}
		return PostgresQueryObjectBuiltin().Fn(forward...)
	}}
}

func prependPostgresReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return builtin.Fn(append([]object.Object{receiver}, args...)...)
	}}
}
func PostgresDatabaseMethod(db *object.PostgresDatabase, name string) object.Object {
	switch name {
	case "exec":
		return prependPostgresReceiver(db, PostgresExecObjectBuiltin())
	case "query":
		return prependPostgresReceiver(db, PostgresQueryObjectBuiltin())
	case "queryOne":
		return prependPostgresReceiver(db, PostgresQueryOneBuiltin())
	case "stream":
		return prependPostgresReceiver(db, PostgresQueryStreamBuiltin())
	case "prepare":
		return prependPostgresReceiver(db, PostgresPrepareBuiltin())
	case "begin":
		return prependPostgresReceiver(db, PostgresBeginBuiltin())
	case "configurePool":
		return prependPostgresReceiver(db, PostgresConfigurePoolBuiltin())
	case "poolStats":
		return prependPostgresReceiver(db, PostgresPoolStatsBuiltin())
	case "ping":
		return prependPostgresReceiver(db, PostgresPingBuiltin())
	case "close":
		return prependPostgresReceiver(db, PostgresCloseBuiltin())
	case "isOpen":
		return prependPostgresReceiver(db, PostgresIsOpenBuiltin())
	default:
		return nil
	}
}
func PostgresStatementMethod(st *object.PostgresStatement, name string) object.Object {
	switch name {
	case "exec":
		return prependPostgresReceiver(st, PostgresStatementExecBuiltin())
	case "query":
		return prependPostgresReceiver(st, PostgresStatementQueryBuiltin())
	case "stream":
		return prependPostgresReceiver(st, PostgresStatementStreamBuiltin())
	case "close":
		return prependPostgresReceiver(st, PostgresStatementCloseBuiltin())
	case "isOpen":
		return prependPostgresReceiver(st, PostgresStatementOpenBuiltin())
	case "sql":
		return prependPostgresReceiver(st, PostgresStatementSQLBuiltin())
	default:
		return nil
	}
}
func PostgresTransactionMethod(tx *object.PostgresTransaction, name string) object.Object {
	switch name {
	case "exec":
		return prependPostgresReceiver(tx, PostgresTransactionExecBuiltin())
	case "query":
		return prependPostgresReceiver(tx, PostgresTransactionQueryBuiltin())
	case "stream":
		return prependPostgresReceiver(tx, PostgresTransactionStreamBuiltin())
	case "prepare":
		return prependPostgresReceiver(tx, PostgresTransactionPrepareBuiltin())
	case "savepoint":
		return prependPostgresReceiver(tx, PostgresSavepointBuiltin())
	case "rollbackTo":
		return prependPostgresReceiver(tx, PostgresRollbackToBuiltin())
	case "release":
		return prependPostgresReceiver(tx, PostgresReleaseBuiltin())
	case "commit":
		return prependPostgresReceiver(tx, PostgresCommitBuiltin())
	case "rollback":
		return prependPostgresReceiver(tx, PostgresRollbackBuiltin())
	case "active":
		return prependPostgresReceiver(tx, PostgresTransactionActiveBuiltin())
	default:
		return nil
	}
}
