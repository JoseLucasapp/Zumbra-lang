package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"
	"zumbra/ast"
	"zumbra/code"
)

type ObjectType string

const (
	HTTP_REQUEST_OBJ         = "HTTP_REQUEST"
	HTTP_RESPONSE_OBJ        = "HTTP_RESPONSE"
	HTTP_APP_OBJ             = "HTTP_APP"
	HTTP_SERVER_OBJ          = "HTTP_SERVER"
	HTTP_CLIENT_RESPONSE_OBJ = "HTTP_CLIENT_RESPONSE"
	HTTP_STREAM_OBJ          = "HTTP_STREAM"
	HTTP_FILE_OBJ            = "HTTP_FILE"
	WEB_SOCKET_OBJ           = "WEB_SOCKET"
	INTEGER_OBJ              = "INTEGER"
	BOOLEAN_OBJ              = "BOOLEAN"
	NULL_OBJ                 = "NULL"
	RETURN_VALUE_OBJ         = "RETURN_VALUE"
	ERROR_OBJ                = "ERROR"
	FUNCTION_OBJ             = "FUNCTION"
	STRING_OBJ               = "STRING"
	BUILTIN_OBJ              = "BUILTIN"
	ARRAY_OBJ                = "ARRAY"
	DICT_OBJ                 = "DICT"
	COMPILED_FUNCTION_OBJ    = "COMPILED_FUNCTION_OBJ"
	CLOSURE_OBJ              = "CLOSURE_OBJ"
	FLOAT_OBJ                = "FLOAT"
	DATE_OBJ                 = "DATE"
	RECORD_OBJ               = "RECORD"
	ENV_OBJ                  = "ENV"
	STRUCT_DEF_OBJ           = "STRUCT_DEFINITION"
	STRUCT_INSTANCE_OBJ      = "STRUCT_INSTANCE"
	BOUND_METHOD_OBJ         = "BOUND_METHOD"
	ENUM_DEF_OBJ             = "ENUM_DEFINITION"
	ENUM_VALUE_OBJ           = "ENUM_VALUE"
	EXTERNAL_FUNCTION_OBJ    = "EXTERNAL_FUNCTION"
	TASK_OBJ                 = "TASK"
	CHANNEL_OBJ              = "CHANNEL"
	MUTEX_OBJ                = "MUTEX"
	RW_MUTEX_OBJ             = "RW_MUTEX"
	WAIT_GROUP_OBJ           = "WAIT_GROUP"
	SEMAPHORE_OBJ            = "SEMAPHORE"
	ATOMIC_INT_OBJ           = "ATOMIC_INT"
	NET_LISTENER_OBJ         = "NET_LISTENER"
	NET_STREAM_OBJ           = "NET_STREAM"
	UDP_SOCKET_OBJ           = "UDP_SOCKET"
	SQLITE_DATABASE_OBJ      = "SQLITE_DATABASE"
	SQLITE_STATEMENT_OBJ     = "SQLITE_STATEMENT"
	SQLITE_TRANSACTION_OBJ   = "SQLITE_TRANSACTION"
	SQL_ROWS_OBJ             = "SQL_ROWS"
	POSTGRES_DATABASE_OBJ    = "POSTGRES_DATABASE"
	POSTGRES_STATEMENT_OBJ   = "POSTGRES_STATEMENT"
	POSTGRES_TRANSACTION_OBJ = "POSTGRES_TRANSACTION"
	REDIS_CLIENT_OBJ         = "REDIS_CLIENT"
	CONFIG_OBJ               = "CONFIG"
	LOGGER_OBJ               = "LOGGER"
	METRICS_OBJ              = "METRICS"
	TRACE_SPAN_OBJ           = "TRACE_SPAN"
	SESSION_STORE_OBJ        = "SESSION_STORE"
	RATE_LIMITER_OBJ         = "RATE_LIMITER"
	DESKTOP_APP_OBJ          = "DESKTOP_APP"
	DESKTOP_WINDOW_OBJ       = "DESKTOP_WINDOW"
	DESKTOP_TRAY_OBJ         = "DESKTOP_TRAY"
	DESKTOP_PROCESS_OBJ      = "DESKTOP_PROCESS"
	UI_NODE_OBJ              = "UI_NODE"
	UI_STATE_OBJ             = "UI_STATE"
	UI_THEME_OBJ             = "UI_THEME"
	UI_CONTEXT_OBJ           = "UI_CONTEXT"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return fmt.Sprintf("ERROR: %s", e.Message) }

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
	Async      bool
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	out.WriteString("fct")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")
	return out.String()
}

type ExternalFunction struct{ Name string }

func (f *ExternalFunction) Type() ObjectType { return EXTERNAL_FUNCTION_OBJ }
func (f *ExternalFunction) Inspect() string  { return "extern C " + f.Name }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range a.Elements {
		elements = append(elements, el.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type DictKey struct {
	Type  ObjectType
	Value uint64
}

func (b *Boolean) DictKey() DictKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}

	return DictKey{Type: b.Type(), Value: value}
}

func (i *Integer) DictKey() DictKey {
	return DictKey{Type: i.Type(), Value: uint64(i.Value)}
}

func (s *String) DictKey() DictKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))

	return DictKey{Type: s.Type(), Value: h.Sum64()}
}

type DictPair struct {
	Key   Object
	Value Object
}

type Dict struct {
	Pairs map[DictKey]DictPair
}

func (d *Dict) Type() ObjectType { return DICT_OBJ }
func (d *Dict) Inspect() string {
	var out bytes.Buffer

	pairs := []string{}

	for _, pair := range d.Pairs {
		pairs = append(pairs, pair.Key.Inspect()+":"+pair.Value.Inspect())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

type Dictable interface {
	DictKey() DictKey
}

type CompiledFunction struct {
	Instructions  code.Instructions
	NumLocals     int
	NumParameters int
	Async         bool
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", cf)
}

type Closure struct {
	Fn   *CompiledFunction
	Free []Object
}

func (c *Closure) Type() ObjectType { return CLOSURE_OBJ }
func (c *Closure) Inspect() string {
	return fmt.Sprintf("Closure[%p]", c)
}

type Float struct {
	Value float64
}

func (b *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string {
	return strconv.FormatFloat(f.Value, 'f', -1, 64)
}

type Date struct {
	FullDate time.Time
	Hour     int
	Minute   int
	Day      int
	Second   int
	Month    time.Month
	Year     int
}

func (d *Date) Type() ObjectType { return DATE_OBJ }
func (d *Date) Inspect() string {
	return d.FullDate.String()
}

type Record struct {
	Fields map[string]interface{}
}

func (r *Record) Type() ObjectType { return RECORD_OBJ }
func (r *Record) Inspect() string {
	return fmt.Sprintf("%v", r.Fields)
}
