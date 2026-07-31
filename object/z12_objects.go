package object

import (
	"fmt"
	"sync"
	"time"
)

// SQLRowsRuntime is the portable streaming row cursor shared by SQLite and PostgreSQL.
type SQLRowsRuntime interface {
	Next() (map[string]Object, bool, error)
	Columns() []string
	Close() error
	IsOpen() bool
}

type SQLRows struct {
	Runtime SQLRowsRuntime
	Driver  string
}

func (r *SQLRows) Type() ObjectType { return SQL_ROWS_OBJ }
func (r *SQLRows) Inspect() string {
	if r == nil || r.Runtime == nil {
		return "SQLRows<nil>"
	}
	return fmt.Sprintf("SQLRows<%s open=%t>", r.Driver, r.Runtime.IsOpen())
}

// PostgreSQL portable runtime contracts.
type PostgresDatabaseRuntime interface {
	Exec(query string, params []Object) (SQLiteExecResult, error)
	Query(query string, params []Object) ([]map[string]Object, error)
	QueryStream(query string, params []Object) (SQLRowsRuntime, error)
	Prepare(query string) (PostgresStatementRuntime, error)
	Begin() (PostgresTransactionRuntime, error)
	Ping() error
	Close() error
	IsOpen() bool
	PoolStats() map[string]int64
	ConfigurePool(maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration)
}

type PostgresStatementRuntime interface {
	Exec(params []Object) (SQLiteExecResult, error)
	Query(params []Object) ([]map[string]Object, error)
	QueryStream(params []Object) (SQLRowsRuntime, error)
	Close() error
	IsOpen() bool
	SQL() string
}

type PostgresTransactionRuntime interface {
	Exec(query string, params []Object) (SQLiteExecResult, error)
	Query(query string, params []Object) ([]map[string]Object, error)
	QueryStream(query string, params []Object) (SQLRowsRuntime, error)
	Prepare(query string) (PostgresStatementRuntime, error)
	Savepoint(name string) error
	RollbackTo(name string) error
	Release(name string) error
	Commit() error
	Rollback() error
	Active() bool
}

type PostgresDatabase struct {
	Runtime PostgresDatabaseRuntime
	DSN     string
}

func (d *PostgresDatabase) Type() ObjectType { return POSTGRES_DATABASE_OBJ }
func (d *PostgresDatabase) Inspect() string {
	if d == nil || d.Runtime == nil {
		return "PostgresDatabase<nil>"
	}
	return fmt.Sprintf("PostgresDatabase<open=%t>", d.Runtime.IsOpen())
}

type PostgresStatement struct{ Runtime PostgresStatementRuntime }

func (s *PostgresStatement) Type() ObjectType { return POSTGRES_STATEMENT_OBJ }
func (s *PostgresStatement) Inspect() string {
	if s == nil || s.Runtime == nil {
		return "PostgresStatement<nil>"
	}
	return fmt.Sprintf("PostgresStatement<open=%t sql=%q>", s.Runtime.IsOpen(), s.Runtime.SQL())
}

type PostgresTransaction struct{ Runtime PostgresTransactionRuntime }

func (t *PostgresTransaction) Type() ObjectType { return POSTGRES_TRANSACTION_OBJ }
func (t *PostgresTransaction) Inspect() string {
	if t == nil || t.Runtime == nil {
		return "PostgresTransaction<nil>"
	}
	return fmt.Sprintf("PostgresTransaction<active=%t>", t.Runtime.Active())
}

// Redis portable runtime contract.
type RedisRuntime interface {
	Ping() error
	Set(key string, value Object, ttl time.Duration) error
	Get(key string) (Object, bool, error)
	Delete(keys []string) (int64, error)
	Exists(keys []string) (int64, error)
	Expire(key string, ttl time.Duration) (bool, error)
	TTL(key string) (time.Duration, error)
	Increment(key string, amount int64) (int64, error)
	Pipeline(commands []RedisCommand) ([]Object, error)
	Close() error
	IsOpen() bool
	PoolStats() map[string]int64
}

type RedisCommand struct {
	Name string
	Args []Object
}

type RedisClient struct {
	Runtime RedisRuntime
	Address string
}

func (c *RedisClient) Type() ObjectType { return REDIS_CLIENT_OBJ }
func (c *RedisClient) Inspect() string {
	if c == nil || c.Runtime == nil {
		return "RedisClient<nil>"
	}
	return fmt.Sprintf("RedisClient<%s open=%t>", c.Address, c.Runtime.IsOpen())
}

// Config stores typed application configuration and tracks secret keys for redaction.
type Config struct {
	Mu      sync.RWMutex
	Values  map[string]Object
	Secrets map[string]bool
}

func (c *Config) Type() ObjectType { return CONFIG_OBJ }
func (c *Config) Inspect() string {
	if c == nil {
		return "Config<nil>"
	}
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return fmt.Sprintf("Config<keys=%d>", len(c.Values))
}

// LoggerRuntime is implemented by the structured logging backend.
type LoggerRuntime interface {
	Log(level, message string, fields map[string]Object) error
	With(fields map[string]Object) LoggerRuntime
	SetLevel(level string) error
	Level() string
	Close() error
}

type Logger struct {
	Runtime LoggerRuntime
	Name    string
}

func (l *Logger) Type() ObjectType { return LOGGER_OBJ }
func (l *Logger) Inspect() string {
	if l == nil || l.Runtime == nil {
		return "Logger<nil>"
	}
	return fmt.Sprintf("Logger<%s level=%s>", l.Name, l.Runtime.Level())
}

// MetricsRuntime stores counters, gauges and histograms.
type MetricsRuntime interface {
	CounterAdd(name string, delta float64, labels map[string]string)
	GaugeSet(name string, value float64, labels map[string]string)
	HistogramObserve(name string, value float64, labels map[string]string)
	Snapshot() map[string]Object
	Reset()
}
type MetricsRegistry struct{ Runtime MetricsRuntime }

func (m *MetricsRegistry) Type() ObjectType { return METRICS_OBJ }
func (m *MetricsRegistry) Inspect() string  { return "MetricsRegistry" }

// TraceSpanRuntime is the portable tracing span contract.
type TraceSpanRuntime interface {
	Child(name string, attributes map[string]Object) TraceSpanRuntime
	SetAttribute(key string, value Object)
	AddEvent(name string, attributes map[string]Object)
	Finish(status string) map[string]Object
	Active() bool
	TraceID() string
	SpanID() string
}
type TraceSpan struct{ Runtime TraceSpanRuntime }

func (s *TraceSpan) Type() ObjectType { return TRACE_SPAN_OBJ }
func (s *TraceSpan) Inspect() string {
	if s == nil || s.Runtime == nil {
		return "TraceSpan<nil>"
	}
	return fmt.Sprintf("TraceSpan<trace=%s span=%s active=%t>", s.Runtime.TraceID(), s.Runtime.SpanID(), s.Runtime.Active())
}

// SessionStoreRuntime provides persistent HTTP-compatible server-side sessions.
type SessionStoreRuntime interface {
	Create(data map[string]Object, ttl time.Duration) (string, error)
	Get(id string) (map[string]Object, bool, error)
	Set(id string, data map[string]Object, ttl time.Duration) error
	Delete(id string) error
	Rotate(id string, ttl time.Duration) (string, error)
	Touch(id string, ttl time.Duration) (bool, error)
	Cleanup() (int64, error)
	Close() error
	IsOpen() bool
}
type SessionStore struct {
	Runtime SessionStoreRuntime
	Driver  string
}

func (s *SessionStore) Type() ObjectType { return SESSION_STORE_OBJ }
func (s *SessionStore) Inspect() string {
	if s == nil || s.Runtime == nil {
		return "SessionStore<nil>"
	}
	return fmt.Sprintf("SessionStore<%s open=%t>", s.Driver, s.Runtime.IsOpen())
}

// RateLimiterRuntime is shared by standalone and HTTP middleware use.
type RateLimiterRuntime interface {
	Allow(key string) (allowed bool, remaining int64, retryAfter time.Duration)
	Reset(key string)
}
type RateLimiter struct{ Runtime RateLimiterRuntime }

func (r *RateLimiter) Type() ObjectType { return RATE_LIMITER_OBJ }
func (r *RateLimiter) Inspect() string  { return "RateLimiter" }
